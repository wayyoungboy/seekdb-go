package seekdb

import (
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// EmbeddedProcess manages a seekdb instance running in embedded mode.
// Uses native CGo bindings to libseekdb.so (matching pyseekdb/seekdb-js).
type EmbeddedProcess struct {
	mu       sync.Mutex
	native   *NativeEmbeddedConn
	baseDir  string
	port     int
	started  bool
	stopped  bool
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
func (ep *EmbeddedProcess) Start(timeout time.Duration) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.started {
		return nil
	}

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

	ep.native = native
	ep.started = true

	// Wait for seekdb to be ready on the network port
	if err := ep.waitForReady(timeout); err != nil {
		native.Close()
		ep.started = false
		ep.native = nil
		return fmt.Errorf("seekdb failed to become ready: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the embedded seekdb instance.
func (ep *EmbeddedProcess) Stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.stopped || !ep.started {
		return nil
	}

	if ep.native != nil {
		ep.native.Close()
		ep.native = nil
	}

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
	return ep.started && !ep.stopped && ep.native != nil
}

// Connect opens a MySQL connection to the embedded seekdb instance.
func (ep *EmbeddedProcess) Connect(database string, poolConfig ConnectionPoolConfig) (*sql.DB, error) {
	if !ep.IsRunning() {
		return nil, ErrNotConnected
	}

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/%s", ep.port, database)
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

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/", ep.port)
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

			// Check if native connection is alive
			if ep.native != nil {
				if err := ep.native.Ping(); err == nil {
					return nil
				}
			}

			// Also check TCP connectivity
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

