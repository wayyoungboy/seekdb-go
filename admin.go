package seekdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// AdminClient provides database management operations for seekdb.
// It supports both embedded mode (Linux) and server mode (all platforms).
type AdminClient struct {
	config    AdminConfig
	db        *sql.DB
	connOnce  sync.Once
	connErr   error
	connected bool
	embedded  *EmbeddedProcess // embedded seekdb process (if running)
}

// NewAdminClient creates a new AdminClient with the given configuration.
// The connection is lazily initialized on first use.
func NewAdminClient(config AdminConfig) (*AdminClient, error) {
	if config.Path == "" && config.Host == "" {
		return nil, ErrInvalidConfig
	}
	return &AdminClient{config: config}, nil
}

// ensureConnection lazily initializes the connection.
func (a *AdminClient) ensureConnection() error {
	a.connOnce.Do(func() {
		if a.config.Path != "" {
			a.connErr = a.connectEmbedded()
			if a.connErr == nil {
				a.connected = true
			}
			return
		}
		if a.config.Host != "" {
			a.connErr = a.connectServer()
			if a.connErr == nil {
				a.connected = true
			}
		}
	})
	return a.connErr
}

// connectEmbedded starts the embedded seekdb process and connects to it.
func (a *AdminClient) connectEmbedded() error {
	embCfg := a.config.EmbeddedConfig
	embCfg.BaseDir = a.config.Path

	ep, err := NewEmbeddedProcess(embCfg)
	if err != nil {
		return fmt.Errorf("failed to create embedded process: %w", err)
	}

	if err := ep.Start(60 * time.Second); err != nil {
		return fmt.Errorf("failed to start embedded seekdb: %w", err)
	}

	a.embedded = ep

	db, err := ep.ConnectAdmin(a.config.PoolConfig)
	if err != nil {
		ep.Stop()
		return fmt.Errorf("failed to connect to embedded seekdb: %w", err)
	}

	a.db = db
	return nil
}

// connectServer establishes the server connection.
func (a *AdminClient) connectServer() error {
	password := a.config.Password
	if password == "" {
		password = os.Getenv("SEEKDB_PASSWORD")
	}

	port := a.config.Port
	if port == 0 {
		port = 2881
	}

	user := a.config.User
	if user == "" {
		user = "root"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, a.config.Host, port)
	if a.config.Tenant != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/?tenant=%s", user, password, a.config.Host, port, a.config.Tenant)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Apply pool configuration
	poolConfig := a.config.PoolConfig
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
		return fmt.Errorf("failed to connect: %w", err)
	}

	a.db = db
	return nil
}

// Close closes the AdminClient connection.
// In embedded mode, this also stops the seekdb subprocess.
func (a *AdminClient) Close() error {
	if a.db != nil {
		a.db.Close()
		a.db = nil
	}
	if a.embedded != nil {
		return a.embedded.Stop()
	}
	return nil
}

// CreateDatabase creates a new database with the given name.
func (a *AdminClient) CreateDatabase(ctx context.Context, name string) error {
	if err := a.ensureConnection(); err != nil {
		return err
	}
	if name == "" {
		return ErrDatabaseNameEmpty
	}

	query := fmt.Sprintf("CREATE DATABASE `%s`", name)
	_, err := a.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	return nil
}

// GetDatabase retrieves information about a specific database.
func (a *AdminClient) GetDatabase(ctx context.Context, name string) (*DatabaseInfo, error) {
	if err := a.ensureConnection(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, ErrDatabaseNameEmpty
	}

	query := "SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?"
	row := a.db.QueryRowContext(ctx, query, name)

	var info DatabaseInfo
	err := row.Scan(&info.Name, &info.Charset, &info.Collation)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDatabaseNotFound
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &info, nil
}

// ListDatabases retrieves a paginated list of databases.
func (a *AdminClient) ListDatabases(ctx context.Context, limit, offset int) ([]DatabaseInfo, error) {
	if err := a.ensureConnection(); err != nil {
		return nil, err
	}
	query := "SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA ORDER BY SCHEMA_NAME LIMIT ? OFFSET ?"
	rows, err := a.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var info DatabaseInfo
		if err := rows.Scan(&info.Name, &info.Charset, &info.Collation); err != nil {
			return nil, fmt.Errorf("failed to scan database: %w", err)
		}
		databases = append(databases, info)
	}

	return databases, nil
}

// DeleteDatabase deletes a database with the given name.
func (a *AdminClient) DeleteDatabase(ctx context.Context, name string) error {
	if err := a.ensureConnection(); err != nil {
		return err
	}
	if name == "" {
		return ErrDatabaseNameEmpty
	}

	query := fmt.Sprintf("DROP DATABASE `%s`", name)
	_, err := a.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	return nil
}
