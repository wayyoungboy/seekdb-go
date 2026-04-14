package seekdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

// Collection table name constants (aligned with pyseekdb)
const (
	collectionTablePrefix    = "c$v1$"
	collectionTableSeparator = "$"
)

// collectionTableName converts a collection name to the internal table name.
func collectionTableName(name string) string {
	return collectionTablePrefix + name
}

// parseCollectionName extracts the collection name from an internal table name.
func parseCollectionName(tableName string) string {
	if len(tableName) > len(collectionTablePrefix) &&
	   tableName[:len(collectionTablePrefix)] == collectionTablePrefix {
		return tableName[len(collectionTablePrefix):]
	}
	return tableName
}

// Client provides collection and data operations for seekdb.
// It supports both embedded mode (Linux) and server mode (all platforms).
type Client struct {
	config    ClientConfig
	db        *sql.DB
	database  string
	connOnce  sync.Once
	connErr   error
	connected bool
}

// NewClient creates a new Client with the given configuration.
// The connection is lazily initialized on first use.
func NewClient(config ClientConfig) (*Client, error) {
	if config.Path == "" && config.Host == "" {
		return nil, ErrInvalidConfig
	}
	return &Client{config: config}, nil
}

// ensureConnection lazily initializes the connection.
func (c *Client) ensureConnection() error {
	c.connOnce.Do(func() {
		if c.config.Path != "" {
			c.connErr = ErrEmbeddedNotSupported
			return
		}
		if c.config.Host != "" {
			c.connErr = c.connectServer()
			if c.connErr == nil {
				c.connected = true
			}
		}
	})
	return c.connErr
}

// connectServer establishes the server connection.
func (c *Client) connectServer() error {
	password := c.config.Password
	if password == "" {
		password = os.Getenv("SEEKDB_PASSWORD")
	}

	port := c.config.Port
	if port == 0 {
		port = 2881
	}

	user := c.config.User
	if user == "" {
		user = "root"
	}

	database := c.config.Database
	if database == "" {
		database = "test"
	}
	c.database = database

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, c.config.Host, port, database)
	if c.config.Tenant != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tenant=%s", user, password, c.config.Host, port, database, c.config.Tenant)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Apply pool configuration
	poolConfig := c.config.PoolConfig
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

	c.db = db
	return nil
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
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
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
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
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
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
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
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
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
		collections = append(collections, CollectionInfo{Name: parseCollectionName(name)})
	}

	return collections, nil
}

// CountCollections returns the number of collections in the database.
func (c *Client) CountCollections(ctx context.Context) (int, error) {
	if err := c.ensureConnection(); err != nil {
		return 0, err
	}
	query := "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME LIKE ?"
	row := c.db.QueryRowContext(ctx, query, c.database, collectionTablePrefix+"%")

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count collections: %w", err)
	}

	return count, nil
}

// HasCollection checks if a collection exists in the database.
func (c *Client) HasCollection(ctx context.Context, name string) (bool, error) {
	if err := c.ensureConnection(); err != nil {
		return false, err
	}
	tableName := collectionTableName(name)
	query := "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	row := c.db.QueryRowContext(ctx, query, c.database, tableName)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection: %w", err)
	}

	return count > 0, nil
}

// DeleteCollection deletes a collection from the database.
func (c *Client) DeleteCollection(ctx context.Context, name string) error {
	if err := c.ensureConnection(); err != nil {
		return err
	}
	if name == "" {
		return ErrCollectionNameEmpty
	}

	tableName := collectionTableName(name)
	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
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

	tableName := collectionTableName(name)
	return fmt.Sprintf(
		"CREATE TABLE `%s` (id VARCHAR(512) PRIMARY KEY, document TEXT, embedding VECTOR(%d), metadata JSON, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
		tableName, dim)
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

	tableName := collectionTableName(name)
	return fmt.Sprintf(
		"CREATE VECTOR INDEX idx_embedding ON `%s` (embedding) WITH (distance = %s, type = %s)",
		tableName, distance, indexType)
}
