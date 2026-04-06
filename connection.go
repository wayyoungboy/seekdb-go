package seekdb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectionMode represents the connection mode for seekdb.
type ConnectionMode string

const (
	ConnectionModeEmbedded ConnectionMode = "embedded"
	ConnectionModeServer   ConnectionMode = "server"
)

// Connection represents a database connection to seekdb.
type Connection struct {
	db     *sql.DB
	mode   ConnectionMode
	config interface{}
}

// NewConnection creates a new connection based on the configuration.
func NewConnection(config interface{}) (*Connection, error) {
	switch c := config.(type) {
	case ClientConfig:
		return newClientConnection(c)
	case AdminConfig:
		return newAdminConnection(c)
	default:
		return nil, ErrInvalidConfig
	}
}

// newClientConnection creates a connection for Client operations.
func newClientConnection(config ClientConfig) (*Connection, error) {
	if config.Path != "" {
		// Embedded mode
		return newEmbeddedConnection(config.Path)
	}
	if config.Host != "" {
		// Server mode
		return newServerConnection(config.Host, config.Port, config.User, config.Password, config.Tenant, config.Database)
	}
	return nil, ErrInvalidConfig
}

// newAdminConnection creates a connection for Admin operations.
func newAdminConnection(config AdminConfig) (*Connection, error) {
	if config.Path != "" {
		// Embedded mode
		return newEmbeddedConnection(config.Path)
	}
	if config.Host != "" {
		// Server mode
		return newServerConnection(config.Host, config.Port, config.User, config.Password, config.Tenant, "")
	}
	return nil, ErrInvalidConfig
}

// newEmbeddedConnection creates an embedded mode connection.
// Note: Embedded mode is only supported on Linux with glibc >= 2.28.
func newEmbeddedConnection(path string) (*Connection, error) {
	// TODO: Implement embedded mode
	// This requires bundling the seekdb binary for Linux
	return nil, ErrEmbeddedNotSupported
}

// newServerConnection creates a server mode connection.
func newServerConnection(host string, port int, user, password, tenant, database string) (*Connection, error) {
	// Apply defaults
	if port == 0 {
		port = 2881
	}
	if user == "" {
		user = "root"
	}
	if password == "" {
		password = os.Getenv("SEEKDB_PASSWORD")
	}

	// Build DSN
	dsn := buildDSN(host, port, user, password, tenant, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Connection{
		db:   db,
		mode: ConnectionModeServer,
	}, nil
}

// buildDSN builds the MySQL DSN string.
func buildDSN(host string, port int, user, password, tenant, database string) string {
	var dsn string

	if database != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	}

	// Add tenant parameter for OceanBase
	if tenant != "" {
		if strings.Contains(dsn, "?") {
			dsn += fmt.Sprintf("&tenant=%s", tenant)
		} else {
			dsn += fmt.Sprintf("?tenant=%s", tenant)
		}
	}

	return dsn
}

// Close closes the connection.
func (c *Connection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DB returns the underlying sql.DB connection.
func (c *Connection) DB() *sql.DB {
	return c.db
}

// Mode returns the connection mode.
func (c *Connection) Mode() ConnectionMode {
	return c.mode
}

// Ping verifies the connection is still alive.
func (c *Connection) Ping() error {
	if c.db == nil {
		return ErrNotConnected
	}
	return c.db.Ping()
}

// IsConnected returns true if the connection is active.
func (c *Connection) IsConnected() bool {
	if c.db == nil {
		return false
	}
	return c.db.Ping() == nil
}