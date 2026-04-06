package seekdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Client provides collection and data operations for seekdb.
// It supports both embedded mode (Linux) and server mode (all platforms).
type Client struct {
	config   ClientConfig
	db       *sql.DB
	database string
}

// NewClient creates a new Client with the given configuration.
// The client automatically selects the appropriate connection mode based on the config:
// - If Path is set: embedded mode (Linux only)
// - If Host is set: server mode (all platforms)
func NewClient(config ClientConfig) (*Client, error) {
	if config.Path != "" {
		// Embedded mode - Linux only
		return newEmbeddedClient(config)
	}
	if config.Host != "" {
		// Server mode
		return newServerClient(config)
	}
	return nil, ErrInvalidConfig
}

// newEmbeddedClient creates a Client for embedded mode.
func newEmbeddedClient(config ClientConfig) (*Client, error) {
	// TODO: Implement embedded mode connection
	// This requires bundling seekdb binary for Linux
	return nil, ErrEmbeddedNotSupported
}

// newServerClient creates a Client for server mode.
func newServerClient(config ClientConfig) (*Client, error) {
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

	database := config.Database
	if database == "" {
		database = "test"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, config.Host, port, database)
	if config.Tenant != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tenant=%s", user, password, config.Host, port, database, config.Tenant)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		config:   config,
		db:       db,
		database: database,
	}, nil
}

// Close closes the Client connection.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// CreateCollection creates a new collection with the given name and configuration.
func (c *Client) CreateCollection(ctx context.Context, name string, config CollectionConfig) (*Collection, error) {
	if name == "" {
		return nil, ErrCollectionNameEmpty
	}

	// Create the table with vector column
	query := c.buildCreateTableSQL(name, config)
	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Create vector index if configured
	if config.VectorIndex != nil {
		indexQuery := c.buildCreateIndexSQL(name, config)
		_, err = c.db.ExecContext(ctx, indexQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
	}

	return &Collection{
		name:   name,
		client: c,
		config: config,
	}, nil
}

// GetCollection retrieves an existing collection by name.
func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error) {
	if name == "" {
		return nil, ErrCollectionNameEmpty
	}

	// Check if the collection exists
	exists, err := c.HasCollection(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCollectionNotFound
	}

	return &Collection{
		name:   name,
		client: c,
	}, nil
}

// GetOrCreateCollection creates a collection if it doesn't exist, or returns the existing one.
func (c *Client) GetOrCreateCollection(ctx context.Context, name string, config CollectionConfig) (*Collection, error) {
	exists, err := c.HasCollection(ctx, name)
	if err != nil {
		return nil, err
	}

	if exists {
		return c.GetCollection(ctx, name)
	}

	return c.CreateCollection(ctx, name, config)
}

// ListCollections retrieves a list of all collections in the database.
func (c *Client) ListCollections(ctx context.Context) ([]CollectionInfo, error) {
	query := "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?"
	rows, err := c.db.QueryContext(ctx, query, c.database)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	var collections []CollectionInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		collections = append(collections, CollectionInfo{Name: name})
	}

	return collections, nil
}

// CountCollections returns the number of collections in the database.
func (c *Client) CountCollections(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?"
	row := c.db.QueryRowContext(ctx, query, c.database)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count collections: %w", err)
	}

	return count, nil
}

// HasCollection checks if a collection exists in the database.
func (c *Client) HasCollection(ctx context.Context, name string) (bool, error) {
	query := "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	row := c.db.QueryRowContext(ctx, query, c.database, name)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection: %w", err)
	}

	return count > 0, nil
}

// DeleteCollection deletes a collection from the database.
func (c *Client) DeleteCollection(ctx context.Context, name string) error {
	if name == "" {
		return ErrCollectionNameEmpty
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", name)
	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// buildCreateTableSQL generates the SQL for creating a collection table.
func (c *Client) buildCreateTableSQL(name string, config CollectionConfig) string {
	dim := config.Dimension
	if dim == 0 {
		dim = 128 // default dimension
	}

	return fmt.Sprintf(
		"CREATE TABLE `%s` (id VARCHAR(512) PRIMARY KEY, document TEXT, embedding VECTOR(%d), metadata JSON, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
		name, dim)
}

// buildCreateIndexSQL generates the SQL for creating a vector index.
func (c *Client) buildCreateIndexSQL(name string, config CollectionConfig) string {
	distance := config.DistanceMetric
	if distance == "" {
		distance = DistanceCosine
	}

	indexType := "HNSW"
	if config.VectorIndex != nil {
		indexType = config.VectorIndex.Type()
	}

	return fmt.Sprintf(
		"CREATE VECTOR INDEX idx_embedding ON `%s` (embedding) WITH (distance = '%s', type = '%s')",
		name, distance, indexType)
}
