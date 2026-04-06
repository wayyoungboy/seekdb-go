package seekdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// AdminClient provides database management operations for seekdb.
// It supports both embedded mode (Linux) and server mode (all platforms).
type AdminClient struct {
	config AdminConfig
	db     *sql.DB
}

// NewAdminClient creates a new AdminClient with the given configuration.
// The client automatically selects the appropriate connection mode based on the config:
// - If Path is set: embedded mode (Linux only)
// - If Host is set: server mode (all platforms)
func NewAdminClient(config AdminConfig) (*AdminClient, error) {
	if config.Path != "" {
		// Embedded mode - Linux only
		return newEmbeddedAdminClient(config)
	}
	if config.Host != "" {
		// Server mode
		return newServerAdminClient(config)
	}
	return nil, ErrInvalidConfig
}

// newEmbeddedAdminClient creates an AdminClient for embedded mode.
func newEmbeddedAdminClient(config AdminConfig) (*AdminClient, error) {
	// TODO: Implement embedded mode connection
	// This requires bundling seekdb binary for Linux
	return nil, ErrEmbeddedNotSupported
}

// newServerAdminClient creates an AdminClient for server mode.
func newServerAdminClient(config AdminConfig) (*AdminClient, error) {
	password := config.Password
	if password == "" {
		password = os.Getenv("SEEKDB_PASSWORD")
	}

	port := config.Port
	if port == 0 {
		port = 2881
	}

	user := config.User
	if user == "" {
		user = "root"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, config.Host, port)
	if config.Tenant != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/?tenant=%s", user, password, config.Host, port, config.Tenant)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &AdminClient{
		config: config,
		db:     db,
	}, nil
}

// Close closes the AdminClient connection.
func (a *AdminClient) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// CreateDatabase creates a new database with the given name.
func (a *AdminClient) CreateDatabase(ctx context.Context, name string) error {
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
