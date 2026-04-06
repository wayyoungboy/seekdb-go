package seekdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Collection represents a collection (table) in seekdb.
type Collection struct {
	name   string
	client *Client
	config CollectionConfig
}

// CollectionConfig holds the configuration for creating a collection.
type CollectionConfig struct {
	Name              string
	Dimension         int               // Vector dimension (default: 128)
	DistanceMetric    DistanceMetric    // Distance metric (default: cosine)
	VectorIndex       VectorIndexConfig // Vector index configuration
	EmbeddingFunction EmbeddingFunction // Optional embedding function
	Metadata          map[string]string // Collection metadata
}

// Name returns the collection name.
func (c *Collection) Name() string {
	return c.tableName()
}

// tableName returns the internal table name with prefix.
func (c *Collection) tableName() string {
	return collectionTableName(c.tableName())
}

// Add inserts new documents into the collection.
func (c *Collection) Add(ctx context.Context, params AddParams) error {
	if len(params.IDs) == 0 {
		return ErrIDRequired
	}

	// If embeddings not provided and no embedding function, error
	if len(params.Embeddings) == 0 && c.config.EmbeddingFunction == nil {
		if len(params.Documents) == 0 {
			return ErrEmbeddingRequired
		}
	}

	// Generate embeddings if not provided
	embeddings := params.Embeddings
	if len(embeddings) == 0 && c.config.EmbeddingFunction != nil && len(params.Documents) > 0 {
		generated, err := c.config.EmbeddingFunction.EmbedDocuments(ctx, params.Documents)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}
		embeddings = generated
	}

	// Validate embedding dimensions
	for i, emb := range embeddings {
		if len(emb) != c.config.Dimension {
			return fmt.Errorf("%w: expected %d, got %d at index %d",
				ErrInvalidEmbedding, c.config.Dimension, len(emb), i)
		}
	}

	// Insert documents
	for i, id := range params.IDs {
		doc := ""
		if i < len(params.Documents) {
			doc = params.Documents[i]
		}

		emb := embeddings[i]
		meta := "{}"
		if i < len(params.Metadatas) {
			meta = metadataToJSON(params.Metadatas[i])
		}

		query := fmt.Sprintf(
			"INSERT INTO `%s` (id, document, embedding, metadata) VALUES (?, ?, ?, ?)",
			c.tableName())

		_, err := c.client.db.ExecContext(ctx, query, id, doc, vectorToSQL(emb), meta)
		if err != nil {
			return fmt.Errorf("failed to add document %s: %w", id, err)
		}
	}

	return nil
}

// AddParams holds the parameters for adding documents.
type AddParams struct {
	IDs        []string
	Documents  []string
	Embeddings [][]float32
	Metadatas  []map[string]interface{}
}

// Update updates existing documents in the collection.
func (c *Collection) Update(ctx context.Context, params UpdateParams) error {
	if len(params.IDs) == 0 {
		return ErrIDRequired
	}

	for i, id := range params.IDs {
		updates := []string{}
		args := []interface{}{}

		if i < len(params.Documents) && params.Documents[i] != "" {
			updates = append(updates, "document = ?")
			args = append(args, params.Documents[i])
		}

		if i < len(params.Embeddings) && len(params.Embeddings[i]) > 0 {
			updates = append(updates, "embedding = ?")
			args = append(args, vectorToSQL(params.Embeddings[i]))
		}

		if i < len(params.Metadatas) {
			updates = append(updates, "metadata = ?")
			args = append(args, metadataToJSON(params.Metadatas[i]))
		}

		if len(updates) == 0 {
			continue
		}

		args = append(args, id)
		query := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?", c.tableName(), joinUpdates(updates))
		_, err := c.client.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to update document %s: %w", id, err)
		}
	}

	return nil
}

// UpdateParams holds the parameters for updating documents.
type UpdateParams struct {
	IDs        []string
	Documents  []string
	Embeddings [][]float32
	Metadatas  []map[string]interface{}
}

// Upsert inserts new documents or updates existing ones.
func (c *Collection) Upsert(ctx context.Context, params UpsertParams) error {
	for i, id := range params.IDs {
		// Check if exists
		var exists bool
		query := fmt.Sprintf("SELECT COUNT(*) > 0 FROM `%s` WHERE id = ?", c.tableName())
		row := c.client.db.QueryRowContext(ctx, query, id)
		if err := row.Scan(&exists); err != nil {
			return fmt.Errorf("failed to check document %s: %w", id, err)
		}

		if exists {
			// Update
			updateParams := UpdateParams{
				IDs:        []string{id},
				Documents:  []string{params.Documents[i]},
				Embeddings: [][]float32{params.Embeddings[i]},
				Metadatas:  []map[string]interface{}{params.Metadatas[i]},
			}
			if err := c.Update(ctx, updateParams); err != nil {
				return err
			}
		} else {
			// Add
			addParams := AddParams{
				IDs:        []string{id},
				Documents:  []string{params.Documents[i]},
				Embeddings: [][]float32{params.Embeddings[i]},
				Metadatas:  []map[string]interface{}{params.Metadatas[i]},
			}
			if err := c.Add(ctx, addParams); err != nil {
				return err
			}
		}
	}

	return nil
}

// UpsertParams holds the parameters for upserting documents.
type UpsertParams struct {
	IDs        []string
	Documents  []string
	Embeddings [][]float32
	Metadatas  []map[string]interface{}
}

// Delete removes documents from the collection.
func (c *Collection) Delete(ctx context.Context, params DeleteParams) error {
	if len(params.IDs) > 0 {
		// Delete by IDs
		for _, id := range params.IDs {
			query := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", c.tableName())
			_, err := c.client.db.ExecContext(ctx, query, id)
			if err != nil {
				return fmt.Errorf("failed to delete document %s: %w", id, err)
			}
		}
		return nil
	}

	// Delete by filter
	whereClause, args := buildWhereClause(params.Where, params.WhereDocument)
	if whereClause == "" {
		whereClause = "1=1"
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", c.tableName(), whereClause)
	_, err := c.client.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// DeleteParams holds the parameters for deleting documents.
type DeleteParams struct {
	IDs           []string
	Where         map[string]interface{}
	WhereDocument map[string]interface{}
}

// Query performs vector similarity search.
func (c *Collection) Query(ctx context.Context, params QueryParams) (*QueryResult, error) {
	// Handle QueryTexts auto-embedding
	if len(params.QueryEmbeddings) == 0 && len(params.QueryTexts) > 0 {
		if c.config.EmbeddingFunction == nil {
			return nil, ErrEmbeddingRequired
		}
		embeddings, err := c.config.EmbeddingFunction.EmbedDocuments(ctx, params.QueryTexts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
		}
		params.QueryEmbeddings = embeddings
	}

	if len(params.QueryEmbeddings) == 0 {
		return nil, ErrQueryEmbeddingRequired
	}

	// Set default include options if not specified
	include := params.Include
	if !include.Documents && !include.Embeddings && !include.Metadatas && !include.Distances {
		include = DefaultInclude()
	}

	results := &QueryResult{
		IDs:        make([][]string, len(params.QueryEmbeddings)),
		Documents:  make([][]string, len(params.QueryEmbeddings)),
		Embeddings: make([][][]float32, len(params.QueryEmbeddings)),
		Metadatas:  make([][]map[string]interface{}, len(params.QueryEmbeddings)),
		Distances:  make([][]float32, len(params.QueryEmbeddings)),
	}

	nResults := params.NResults
	if nResults == 0 {
		nResults = 10
	}

	for i, queryEmb := range params.QueryEmbeddings {
		whereClause := buildWhereClauseOrDefault(params.Where, params.WhereDocument)

		// Build SELECT fields based on include options
		selectFields := c.buildSelectFields(include, true)

		query := fmt.Sprintf(
			"SELECT %s FROM `%s` WHERE %s ORDER BY VECTOR_DISTANCE(embedding, ?) ASC LIMIT ?",
			selectFields, c.tableName(), whereClause)

		args := []interface{}{vectorToSQL(queryEmb)}
		args = append(args, nResults)

		rows, err := c.client.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			var doc, embStr, metaStr sql.NullString
			var distance sql.NullFloat64

			// Scan based on include options
			scanArgs := []interface{}{&id}
			if include.Documents {
				scanArgs = append(scanArgs, &doc)
			}
			if include.Embeddings {
				scanArgs = append(scanArgs, &embStr)
			}
			if include.Metadatas {
				scanArgs = append(scanArgs, &metaStr)
			}
			if include.Distances {
				scanArgs = append(scanArgs, &distance)
			}

			if err := rows.Scan(scanArgs...); err != nil {
				return nil, fmt.Errorf("failed to scan result: %w", err)
			}

			results.IDs[i] = append(results.IDs[i], id)
			if include.Documents {
				results.Documents[i] = append(results.Documents[i], doc.String)
			}
			if include.Embeddings {
				results.Embeddings[i] = append(results.Embeddings[i], parseVector(embStr.String))
			}
			if include.Metadatas {
				results.Metadatas[i] = append(results.Metadatas[i], parseMetadata(metaStr.String))
			}
			if include.Distances {
				results.Distances[i] = append(results.Distances[i], float32(distance.Float64))
			}
		}
	}

	return results, nil
}

// buildSelectFields builds the SELECT field list based on include options.
func (c *Collection) buildSelectFields(include IncludeOptions, withDistance bool) string {
	fields := []string{"id"}
	if include.Documents {
		fields = append(fields, "document")
	}
	if include.Embeddings {
		fields = append(fields, "embedding")
	}
	if include.Metadatas {
		fields = append(fields, "metadata")
	}
	if withDistance && include.Distances {
		fields = append(fields, "VECTOR_DISTANCE(embedding, ?) as distance")
	}
	return strings.Join(fields, ", ")
}

// QueryParams holds the parameters for vector similarity search.
type QueryParams struct {
	QueryEmbeddings [][]float32
	QueryTexts      []string                   // Text queries to be auto-embedded
	NResults        int
	Where           map[string]interface{}
	WhereDocument   map[string]interface{}
	Include         IncludeOptions             // Control which fields to include
}

// Get retrieves documents by ID or filter (non-vector search).
func (c *Collection) Get(ctx context.Context, params GetParams) (*GetResult, error) {
	// Set default include options if not specified
	include := params.Include
	if !include.Documents && !include.Embeddings && !include.Metadatas {
		include = DefaultInclude()
		include.Distances = false // No distances for Get
	}

	if len(params.IDs) > 0 {
		// Get by IDs
		results := &GetResult{}
		for _, id := range params.IDs {
			selectFields := c.buildSelectFields(include, false)
			query := fmt.Sprintf("SELECT %s FROM `%s` WHERE id = ?", selectFields, c.tableName())
			row := c.client.db.QueryRowContext(ctx, query, id)

			var doc, embStr, metaStr sql.NullString
			scanArgs := []interface{}{&id}
			if include.Documents {
				scanArgs = append(scanArgs, &doc)
			}
			if include.Embeddings {
				scanArgs = append(scanArgs, &embStr)
			}
			if include.Metadatas {
				scanArgs = append(scanArgs, &metaStr)
			}

			if err := row.Scan(scanArgs...); err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return nil, fmt.Errorf("failed to get document %s: %w", id, err)
			}
			results.IDs = append(results.IDs, id)
			if include.Documents {
				results.Documents = append(results.Documents, doc.String)
			}
			if include.Embeddings {
				results.Embeddings = append(results.Embeddings, parseVector(embStr.String))
			}
			if include.Metadatas {
				results.Metadatas = append(results.Metadatas, parseMetadata(metaStr.String))
			}
		}
		return results, nil
	}

	// Get by filter
	limit := params.Limit
	if limit == 0 {
		limit = 100
	}

	whereClause, args := buildWhereClause(params.Where, params.WhereDocument)
	if whereClause == "" {
		whereClause = "1=1"
	}

	selectFields := c.buildSelectFields(include, false)
	query := fmt.Sprintf(
		"SELECT %s FROM `%s` WHERE %s LIMIT ? OFFSET ?",
		selectFields, c.tableName(), whereClause)

	args = append(args, limit, params.Offset)
	rows, err := c.client.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	defer rows.Close()

	results := &GetResult{}
	for rows.Next() {
		var id string
		var doc, embStr, metaStr sql.NullString
		scanArgs := []interface{}{&id}
		if include.Documents {
			scanArgs = append(scanArgs, &doc)
		}
		if include.Embeddings {
			scanArgs = append(scanArgs, &embStr)
		}
		if include.Metadatas {
			scanArgs = append(scanArgs, &metaStr)
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results.IDs = append(results.IDs, id)
		if include.Documents {
			results.Documents = append(results.Documents, doc.String)
		}
		if include.Embeddings {
			results.Embeddings = append(results.Embeddings, parseVector(embStr.String))
		}
		if include.Metadatas {
			results.Metadatas = append(results.Metadatas, parseMetadata(metaStr.String))
		}
	}

	return results, nil
}

// GetParams holds the parameters for getting documents.
type GetParams struct {
	IDs           []string
	Where         map[string]interface{}
	WhereDocument map[string]interface{}
	Limit         int
	Offset        int
	Include       IncludeOptions // Control which fields to include
}

// HybridSearchParams holds the parameters for hybrid search.
type HybridSearchParams struct {
	Query    map[string]interface{}
	KNN      map[string]interface{}
	Rank     RankConfig
	NResults int
	Include  IncludeOptions // Control which fields to include
}

// Count returns the number of documents in the collection.
func (c *Collection) Count(ctx context.Context) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", c.tableName())
	row := c.client.db.QueryRowContext(ctx, query)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// Peek returns a preview of the first few documents in the collection.
func (c *Collection) Peek(ctx context.Context, limit int) (*GetResult, error) {
	if limit == 0 {
		limit = 10
	}

	return c.Get(ctx, GetParams{Limit: limit})
}

// Helper functions

func joinUpdates(updates []string) string {
	result := ""
	for i, u := range updates {
		if i > 0 {
			result += ", "
		}
		result += u
	}
	return result
}
