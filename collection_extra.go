package seekdb

import (
	"context"
	"fmt"
	"strings"
)

// ForkOptions holds the options for forking a collection.
type ForkOptions struct {
	Name       string         // Name of the new collection
	Dimension  int            // Optional: override dimension
	Distance   DistanceMetric // Optional: override distance metric
}

// Fork creates a new collection as a copy of the current collection.
// The new collection inherits the schema but starts empty.
func (c *Collection) Fork(ctx context.Context, options ForkOptions) (*Collection, error) {
	if options.Name == "" {
		return nil, ErrCollectionNameEmpty
	}

	// Build new collection config
	newConfig := c.config
	if options.Dimension > 0 {
		newConfig.Dimension = options.Dimension
	}
	if options.Distance != "" {
		newConfig.DistanceMetric = options.Distance
	}

	// Create the new table with the same schema
	createSQL := c.client.buildCreateTableSQL(options.Name, newConfig)
	_, err := c.client.db.ExecContext(ctx, createSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to fork collection: %w", err)
	}

	// Create index if configured
	if newConfig.VectorIndex != nil {
		indexSQL := c.client.buildCreateIndexSQL(options.Name, newConfig)
		_, err = c.client.db.ExecContext(ctx, indexSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to create index for forked collection: %w", err)
		}
	}

	return &Collection{
		name:   options.Name,
		client: c.client,
		config: newConfig,
	}, nil
}

// ModifyOptions holds options for modifying a collection.
type ModifyOptions struct {
	Dimension      int              // New vector dimension
	DistanceMetric DistanceMetric   // New distance metric
	VectorIndex    VectorIndexConfig // New vector index config
}

// Modify updates the collection schema.
// This may require dropping and recreating the collection in some cases.
func (c *Collection) Modify(ctx context.Context, options ModifyOptions) error {
	// Check if any modifications require table changes
	needsRebuild := false

	if options.Dimension > 0 && options.Dimension != c.config.Dimension {
		needsRebuild = true
		c.config.Dimension = options.Dimension
	}

	if options.DistanceMetric != "" && options.DistanceMetric != c.config.DistanceMetric {
		needsRebuild = true
		c.config.DistanceMetric = options.DistanceMetric
	}

	if options.VectorIndex != nil {
		c.config.VectorIndex = options.VectorIndex
	}

	if needsRebuild {
		return fmt.Errorf("dimension or distance metric modification requires recreating the collection - data will be lost. Use Fork to preserve data")
	}

	// If only index changed, we can just recreate the index
	if options.VectorIndex != nil {
		// Drop old index
		dropIndexSQL := fmt.Sprintf("DROP INDEX idx_embedding ON `%s`", c.tableName())
		_, err := c.client.db.ExecContext(ctx, dropIndexSQL)
		if err != nil {
			// Index might not exist, continue
		}

		// Create new index
		createIndexSQL := c.client.buildCreateIndexSQL(c.name, c.config)
		_, err = c.client.db.ExecContext(ctx, createIndexSQL)
		if err != nil {
			return fmt.Errorf("failed to modify index: %w", err)
		}
	}

	return nil
}

// ModifyCollection updates an existing collection's configuration.
func (c *Client) ModifyCollection(ctx context.Context, name string, options ModifyOptions) error {
	collection, err := c.GetCollection(ctx, name)
	if err != nil {
		return err
	}
	return collection.Modify(ctx, options)
}

// ForkCollection creates a new collection as a fork of an existing one.
func (c *Client) ForkCollection(ctx context.Context, sourceName string, options ForkOptions) (*Collection, error) {
	source, err := c.GetCollection(ctx, sourceName)
	if err != nil {
		return nil, err
	}
	return source.Fork(ctx, options)
}

// GetCollectionConfig returns the configuration of a collection.
func (c *Collection) GetConfig() CollectionConfig {
	return c.config
}

// SetEmbeddingFunction sets or updates the embedding function for the collection.
func (c *Collection) SetEmbeddingFunction(fn EmbeddingFunction) {
	c.config.EmbeddingFunction = fn
}

// Validate checks if the collection configuration is valid.
func (c *Collection) Validate() error {
	if c.name == "" {
		return ErrCollectionNameEmpty
	}
	if c.config.Dimension <= 0 {
		return ErrInvalidDimension
	}
	return nil
}

// CollectionDescription holds descriptive information about a collection.
type CollectionDescription struct {
	Name              string
	Dimension         int
	DistanceMetric    DistanceMetric
	IndexType         string
	EmbeddingFunction string
	DocumentCount     int
}

// Describe returns a description of the collection.
func (c *Collection) Describe(ctx context.Context) (*CollectionDescription, error) {
	count, err := c.Count(ctx)
	if err != nil {
		return nil, err
	}

	indexType := ""
	if c.config.VectorIndex != nil {
		indexType = c.config.VectorIndex.Type()
	}

	embeddingFn := ""
	if c.config.EmbeddingFunction != nil {
		embeddingFn = c.config.EmbeddingFunction.Name()
	}

	return &CollectionDescription{
		Name:              c.name,
		Dimension:         c.config.Dimension,
		DistanceMetric:    c.config.DistanceMetric,
		IndexType:         indexType,
		EmbeddingFunction: embeddingFn,
		DocumentCount:     count,
	}, nil
}

// Helper function to check if a string is empty
func isEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}