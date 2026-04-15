// Native embedded connection using CGo bindings to libseekdb.so.
// This replaces the subprocess-based approach with direct C API calls,
// matching how pyseekdb and seekdb-js implement embedded mode.

//go:build cgo && (linux || darwin)

package seekdb

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// NativeEmbeddedConn manages a native seekdb instance via CGo.
type NativeEmbeddedConn struct {
	mu       sync.Mutex
	dbDir    string
	port     int
	opened   bool
	conn     *SeekdbConn
	database string
}

// NativeEmbeddedConfig holds configuration for native embedded mode.
type NativeEmbeddedConfig struct {
	BaseDir string
	Port    int // 0 = no network, > 0 = server mode on port
}

// NewNativeEmbeddedConn creates a new native embedded connection.
func NewNativeEmbeddedConn(cfg NativeEmbeddedConfig) (*NativeEmbeddedConn, error) {
	if cfg.BaseDir == "" {
		return nil, fmt.Errorf("BaseDir is required for native embedded mode")
	}
	return &NativeEmbeddedConn{
		dbDir: cfg.BaseDir,
		port:  cfg.Port,
	}, nil
}

// Open initializes the embedded seekdb instance.
func (n *NativeEmbeddedConn) Open() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.opened {
		return nil
	}

	var err error
	if n.port > 0 {
		err = seekdbOpenWithService(n.dbDir, n.port)
	} else {
		err = seekdbOpen(n.dbDir)
	}
	if err != nil {
		return fmt.Errorf("failed to open seekdb: %w", err)
	}
	n.opened = true
	return nil
}

// Connect opens a connection to the embedded database.
func (n *NativeEmbeddedConn) Connect(database string) error {
	if !n.opened {
		return fmt.Errorf("seekdb not opened")
	}

	conn, err := seekdbConnect(database, true)
	if err != nil {
		return fmt.Errorf("failed to connect to seekdb: %w", err)
	}

	n.mu.Lock()
	n.conn = conn
	n.database = database
	n.mu.Unlock()
	return nil
}

// Conn returns the underlying SeekdbConn.
func (n *NativeEmbeddedConn) Conn() *SeekdbConn {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.conn
}

// Ping checks if the connection is alive.
func (n *NativeEmbeddedConn) Ping() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.conn == nil {
		return fmt.Errorf("not connected")
	}
	return n.conn.Ping()
}

// Close closes the connection and shuts down the embedded instance.
func (n *NativeEmbeddedConn) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
	if n.opened {
		seekdbClose()
		n.opened = false
	}
	return nil
}

// Port returns the port (0 if no network).
func (n *NativeEmbeddedConn) Port() int {
	return n.port
}

// Database returns the current database name.
func (n *NativeEmbeddedConn) Database() string {
	return n.database
}

// Query executes a SQL query and returns rows as [][]string.
func (n *NativeEmbeddedConn) Query(sql string, params ...interface{}) ([][]string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	var result *SeekdbResult
	var err error
	if len(params) > 0 {
		result, err = n.conn.QueryWithParams(sql, params)
	} else {
		result, err = n.conn.Query(sql)
	}
	if err != nil {
		return nil, err
	}
	defer result.Free()

	return result.FetchAll()
}

// Exec executes a SQL statement.
func (n *NativeEmbeddedConn) Exec(sql string, params ...interface{}) (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.conn == nil {
		return 0, fmt.Errorf("not connected")
	}

	if len(params) > 0 {
		return n.conn.ExecWithParams(sql, params)
	}
	return n.conn.Exec(sql)
}

// PoolConfig holds connection pool settings for embedded mode.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ToSQLDB returns a *sql.DB that uses the native embedded connection.
// This is a compatibility wrapper - it creates a fake DSN that the mysql
// driver can't use, so callers should use NativeEmbeddedConn directly.
// For now, embedded mode users should use NativeEmbeddedConn.Query/Exec
// instead of *sql.DB.
func (n *NativeEmbeddedConn) ToSQLDB(cfg PoolConfig) (*sql.DB, error) {
	// The native CGo connection doesn't directly map to *sql.DB.
	// We return a placeholder - the actual query execution goes through
	// NativeEmbeddedConn.Query/Exec.
	// TODO: Implement a database/sql driver for the CGo connection.
	return nil, fmt.Errorf("native embedded mode does not support *sql.DB yet")
}

// WaitForReady polls until the connection responds or timeout.
func (n *NativeEmbeddedConn) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for seekdb to be ready")
			}
			if err := n.Ping(); err == nil {
				return nil
			}
		}
	}
}
