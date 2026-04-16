package seekdb

import (
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// globalSeekdb tracks the singleton seekdb embedded instance state.
// Aligned with pyseekdb: seekdb.open() is called once and never explicitly
// closed. seekdb_close() clears global state and breaks subsequent opens.
var globalSeekdb = struct {
	mu       sync.Mutex
	path     string
	opened   bool
	port     int
	refCount int
}{
	path:     "",
	opened:   false,
	port:     0,
	refCount: 0,
}

// EmbeddedProcess manages a seekdb instance running in embedded mode.
// Uses native CGo bindings to libseekdb.so (matching pyseekdb/seekdb-js).
//
// Aligned with pyseekdb's SeekdbEmbeddedClient: multiple EmbeddedProcess
// instances share the same underlying seekdb engine via a global singleton.
// seekdb.open() is called once; subsequent opens are skipped.
// Close() only decrements the reference count and closes connection handles;
// seekdb_close() is never called to avoid breaking the global state.
type EmbeddedProcess struct {
	mu      sync.Mutex
	baseDir string
	port    int
	started bool
	stopped bool
}

// EmbeddedConfig holds configuration for starting seekdb in embedded mode.
type EmbeddedConfig struct {
	BinaryPath  string            // Deprecated: native CGo mode doesn't use binary. Kept for API compatibility.
	BaseDir     string            // Data directory for seekdb
	Port        int               // Port to listen on (0 = auto-assign)
	ExtraParams map[string]string // Additional parameters (not used in native mode)
	LogLevel    string            // Log level (not used in native mode)
}

// NewEmbeddedProcess creates a new EmbeddedProcess with the given configuration.
func NewEmbeddedProcess(cfg EmbeddedConfig) (*EmbeddedProcess, error) {
	if cfg.BaseDir == "" {
		return nil, fmt.Errorf("BaseDir is required for embedded mode")
	}

	port := cfg.Port
	if port == 0 {
		// Auto-assign a free port
		var err error
		port, err = findFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
	}

	return &EmbeddedProcess{
		baseDir: cfg.BaseDir,
		port:    port,
	}, nil
}

// Start initializes the seekdb embedded instance via CGo.
// If another EmbeddedProcess has already started seekdb, this reuses the existing instance.
func (ep *EmbeddedProcess) Start(timeout time.Duration) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.started {
		return nil
	}

	// Acquire global lock to manage singleton state
	globalSeekdb.mu.Lock()
	defer globalSeekdb.mu.Unlock()

	if !globalSeekdb.opened {
		// First caller: actually open seekdb
		cfg := NativeEmbeddedConfig{
			BaseDir: ep.baseDir,
			Port:    ep.port,
		}

		native, err := NewNativeEmbeddedConn(cfg)
		if err != nil {
			return fmt.Errorf("failed to create native embedded connection: %w", err)
		}

		if err := native.Open(); err != nil {
			return fmt.Errorf("failed to open seekdb: %w", err)
		}

		globalSeekdb.path = ep.baseDir
		globalSeekdb.port = ep.port
		globalSeekdb.opened = true
	} else {
		// Already opened: verify the path matches
		if globalSeekdb.path != ep.baseDir {
			return fmt.Errorf("seekdb already opened with different path: %s (current: %s). "+
				"Multiple embedded instances in the same process are not supported",
				globalSeekdb.path, ep.baseDir)
		}
	}

	globalSeekdb.refCount++
	ep.started = true
	ep.port = globalSeekdb.port

	// Wait for seekdb to be ready on the network port
	if err := ep.waitForReady(timeout); err != nil {
		globalSeekdb.refCount--
		ep.started = false
		return fmt.Errorf("seekdb failed to become ready: %w", err)
	}

	return nil
}

// Stop decrements the reference count. The last caller would need to clean up,
// but we skip seekdb_close() to match pyseekdb behavior (it's never called).
func (ep *EmbeddedProcess) Stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.stopped || !ep.started {
		return nil
	}

	globalSeekdb.mu.Lock()
	globalSeekdb.refCount--
	globalSeekdb.mu.Unlock()

	ep.stopped = true
	ep.started = false
	return nil
}

// Port returns the port the embedded seekdb is listening on.
func (ep *EmbeddedProcess) Port() int {
	return ep.port
}

// BaseDir returns the base directory.
func (ep *EmbeddedProcess) BaseDir() string {
	return ep.baseDir
}

// IsRunning returns true if the instance is running.
func (ep *EmbeddedProcess) IsRunning() bool {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return ep.started && !ep.stopped
}

// IsGlobalRunning returns true if the global seekdb instance is opened.
func IsGlobalRunning() bool {
	globalSeekdb.mu.Lock()
	defer globalSeekdb.mu.Unlock()
	return globalSeekdb.opened
}

// GlobalPort returns the port of the global seekdb instance.
func GlobalPort() int {
	globalSeekdb.mu.Lock()
	defer globalSeekdb.mu.Unlock()
	return globalSeekdb.port
}

// Connect opens a MySQL connection to the embedded seekdb instance.
func (ep *EmbeddedProcess) Connect(database string, poolConfig ConnectionPoolConfig) (*sql.DB, error) {
	if !ep.IsRunning() {
		return nil, ErrNotConnected
	}

	globalSeekdb.mu.Lock()
	port := globalSeekdb.port
	globalSeekdb.mu.Unlock()

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/%s", port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if poolConfig.MaxOpenConns == 0 {
		poolConfig = DefaultConnectionPoolConfig()
	}
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	if poolConfig.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	}
	if poolConfig.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping embedded seekdb: %w", err)
	}

	return db, nil
}

// ConnectAdmin opens a MySQL connection without a database (for admin operations).
func (ep *EmbeddedProcess) ConnectAdmin(poolConfig ConnectionPoolConfig) (*sql.DB, error) {
	if !ep.IsRunning() {
		return nil, ErrNotConnected
	}

	globalSeekdb.mu.Lock()
	port := globalSeekdb.port
	globalSeekdb.mu.Unlock()

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/", port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if poolConfig.MaxOpenConns == 0 {
		poolConfig = DefaultConnectionPoolConfig()
	}
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	if poolConfig.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	}
	if poolConfig.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping embedded seekdb: %w", err)
	}

	return db, nil
}

// waitForReady polls until seekdb responds on the network port or timeout.
func (ep *EmbeddedProcess) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for seekdb to be ready on port %d", ep.port)
			}

			// Check TCP connectivity
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", ep.port), 500*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

// findFreePort finds an available TCP port.
func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}
