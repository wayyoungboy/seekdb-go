package seekdb

import (
	"context"
	"fmt"
)

// HybridSearchResult represents a result from hybrid search with combined scores.
type HybridSearchResult struct {
	ID        string
	Document  string
	Embedding []float32
	Metadata  map[string]interface{}
	Score     float32 // Combined RRF score
	VectorScore float32 // Vector similarity score
	TextScore   float32 // Full-text search score
}

// HybridSearch performs combined full-text and vector similarity search.
// It uses Reciprocal Rank Fusion (RRF) to combine results from both search methods.
func (c *Collection) HybridSearch(ctx context.Context, params HybridSearchParams) (*QueryResult, error) {
	if params.NResults == 0 {
		params.NResults = 10
	}

	// Set default RRF K parameter
	rrfK := params.Rank.RRF.K
	if rrfK == 0 {
		rrfK = 60 // Default RRF K value
	}

	// Perform vector search
	vectorResults, err := c.vectorSearch(ctx, params.KNN, params.NResults*2)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Perform full-text search
	textResults, err := c.textSearch(ctx, params.Query, params.NResults*2)
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}

	// Combine results using RRF
	combinedResults := rrfMerge(vectorResults, textResults, rrfK, params.NResults)

	// Convert to QueryResult format
	result := &QueryResult{
		IDs:        [][]string{combinedResults.IDs},
		Documents:  [][]string{combinedResults.Documents},
		Embeddings: [][][]float32{combinedResults.Embeddings},
		Metadatas:  [][]map[string]interface{}{combinedResults.Metadatas},
		Distances:  [][]float32{combinedResults.Scores},
	}

	return result, nil
}

// vectorSearch performs vector similarity search.
func (c *Collection) vectorSearch(ctx context.Context, knnParams map[string]interface{}, nResults int) (*searchResults, error) {
	// Extract query embeddings from KNN params
	queryEmbeddings, ok := knnParams["query_embeddings"].([][]float32)
	if !ok || len(queryEmbeddings) == 0 {
		return nil, ErrQueryEmbeddingRequired
	}

	// Use the first query embedding
	queryEmb := queryEmbeddings[0]

	results := &searchResults{
		IDs:        []string{},
		Documents:  []string{},
		Embeddings: [][]float32{},
		Metadatas:  []map[string]interface{}{},
		Scores:     []float32{},
	}

	whereClause := "1=1"
	if where, ok := knnParams["where"].(map[string]interface{}); ok {
		clause, _ := buildWhereClause(where, nil)
		if clause != "" {
			whereClause = clause
		}
	}

	query := fmt.Sprintf(
		"SELECT id, document, embedding, metadata, VECTOR_DISTANCE(embedding, ?) as distance FROM `%s` WHERE %s ORDER BY distance ASC LIMIT ?",
		c.tableName(), whereClause)

	args := []interface{}{vectorToSQL(queryEmb), nResults}

	rows, err := c.client.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, doc, embStr, metaStr string
		var distance float32

		if err := rows.Scan(&id, &doc, &embStr, &metaStr, &distance); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		results.IDs = append(results.IDs, id)
		results.Documents = append(results.Documents, doc)
		results.Embeddings = append(results.Embeddings, parseVector(embStr))
		results.Metadatas = append(results.Metadatas, parseMetadata(metaStr))
		results.Scores = append(results.Scores, distance)
	}

	return results, nil
}

// textSearch performs full-text search.
func (c *Collection) textSearch(ctx context.Context, queryParams map[string]interface{}, nResults int) (*searchResults, error) {
	results := &searchResults{
		IDs:        []string{},
		Documents:  []string{},
		Embeddings: [][]float32{},
		Metadatas:  []map[string]interface{}{},
		Scores:     []float32{},
	}

	// Extract search text
	queryText, ok := queryParams["query_text"].(string)
	if !ok || queryText == "" {
		return results, nil // Return empty results if no query text
	}

	whereClause := "1=1"
	if where, ok := queryParams["where"].(map[string]interface{}); ok {
		clause, _ := buildWhereClause(where, nil)
		if clause != "" {
			whereClause = clause
		}
	}

	// Use BM25 for full-text search (seekdb built-in)
	query := fmt.Sprintf(
		"SELECT id, document, embedding, metadata, BM25_SCORE() as score FROM `%s` WHERE MATCH(document) AGAINST(? IN NATURAL LANGUAGE MODE) AND %s ORDER BY score DESC LIMIT ?",
		c.tableName(), whereClause)

	args := []interface{}{queryText, nResults}

	rows, err := c.client.db.QueryContext(ctx, query, args...)
	if err != nil {
		// If BM25 is not available, fall back to LIKE search
		query = fmt.Sprintf(
			"SELECT id, document, embedding, metadata, 1.0 as score FROM `%s` WHERE document LIKE ? AND %s LIMIT ?",
			c.tableName(), whereClause)
		args = []interface{}{fmt.Sprintf("%%%s%%", queryText), nResults}

		rows, err = c.client.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to text search: %w", err)
		}
	}
	defer rows.Close()

	for rows.Next() {
		var id, doc, embStr, metaStr string
		var score float32

		if err := rows.Scan(&id, &doc, &embStr, &metaStr, &score); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		results.IDs = append(results.IDs, id)
		results.Documents = append(results.Documents, doc)
		results.Embeddings = append(results.Embeddings, parseVector(embStr))
		results.Metadatas = append(results.Metadatas, parseMetadata(metaStr))
		results.Scores = append(results.Scores, score)
	}

	return results, nil
}

// searchResults holds internal search results for RRF merging.
type searchResults struct {
	IDs        []string
	Documents  []string
	Embeddings [][]float32
	Metadatas  []map[string]interface{}
	Scores     []float32
}

// rrfMerge combines results from multiple searches using Reciprocal Rank Fusion.
func rrfMerge(vectorResults, textResults *searchResults, k int, nResults int) *searchResults {
	// Build rank maps for each result set
	vectorRanks := make(map[string]int)
	for i, id := range vectorResults.IDs {
		vectorRanks[id] = i + 1 // Rank starts at 1
	}

	textRanks := make(map[string]int)
	for i, id := range textResults.IDs {
		textRanks[id] = i + 1
	}

	// Collect all unique IDs
	allIDs := make(map[string]bool)
	for _, id := range vectorResults.IDs {
		allIDs[id] = true
	}
	for _, id := range textResults.IDs {
		allIDs[id] = true
	}

	// Calculate RRF scores for each document
	rrfScores := make(map[string]float32)
	for id := range allIDs {
		score := float32(0)
		if rank, ok := vectorRanks[id]; ok {
			score += float32(1.0 / (float64(k) + float64(rank)))
		}
		if rank, ok := textRanks[id]; ok {
			score += float32(1.0 / (float64(k) + float64(rank)))
		}
		rrfScores[id] = score
	}

	// Sort by RRF score and build final results
	result := &searchResults{
		IDs:        []string{},
		Documents:  []string{},
		Embeddings: [][]float32{},
		Metadatas:  []map[string]interface{}{},
		Scores:     []float32{},
	}

	// Build lookup maps for document data
	vectorData := make(map[string]struct {
		doc      string
		emb      []float32
		meta     map[string]interface{}
		vecScore float32
	})
	for i, id := range vectorResults.IDs {
		vectorData[id] = struct {
			doc      string
			emb      []float32
			meta     map[string]interface{}
			vecScore float32
		}{
			vectorResults.Documents[i],
			vectorResults.Embeddings[i],
			vectorResults.Metadatas[i],
			vectorResults.Scores[i],
		}
	}

	textData := make(map[string]struct {
		doc      string
		emb      []float32
		meta     map[string]interface{}
		txtScore float32
	})
	for i, id := range textResults.IDs {
		textData[id] = struct {
			doc      string
			emb      []float32
			meta     map[string]interface{}
			txtScore float32
		}{
			textResults.Documents[i],
			textResults.Embeddings[i],
			textResults.Metadatas[i],
			textResults.Scores[i],
		}
	}

	// Create sorted list by RRF score
	type scoreEntry struct {
		id    string
		score float32
	}
	sortedScores := make([]scoreEntry, 0, len(rrfScores))
	for id, score := range rrfScores {
		sortedScores = append(sortedScores, scoreEntry{id, score})
	}

	// Sort descending by score
	for i := 0; i < len(sortedScores); i++ {
		for j := i + 1; j < len(sortedScores); j++ {
			if sortedScores[j].score > sortedScores[i].score {
				sortedScores[i], sortedScores[j] = sortedScores[j], sortedScores[i]
			}
		}
	}

	// Build final results (limited to nResults)
	count := 0
	for _, entry := range sortedScores {
		if count >= nResults {
			break
		}

		id := entry.id

		// Get document data (prefer vector data, fallback to text data)
		var doc string
		var emb []float32
		var meta map[string]interface{}

		if data, ok := vectorData[id]; ok {
			doc = data.doc
			emb = data.emb
			meta = data.meta
		} else if data, ok := textData[id]; ok {
			doc = data.doc
			emb = data.emb
			meta = data.meta
		}

		result.IDs = append(result.IDs, id)
		result.Documents = append(result.Documents, doc)
		result.Embeddings = append(result.Embeddings, emb)
		result.Metadatas = append(result.Metadatas, meta)
		result.Scores = append(result.Scores, entry.score)

		count++
	}

	return result
}

// RRF calculation helper
func rrfScore(rank int, k int) float32 {
	return float32(1.0 / (float64(k) + float64(rank)))
}

// CreateFulltextIndex creates a full-text index on the document column.
func (c *Collection) CreateFulltextIndex(ctx context.Context) error {
	query := fmt.Sprintf("CREATE FULLTEXT INDEX idx_document_ft ON `%s` (document)", c.tableName())
	_, err := c.client.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create fulltext index: %w", err)
	}
	return nil
}

// HybridSearchWithText performs hybrid search with a text query (auto-embeds if needed).
func (c *Collection) HybridSearchWithText(ctx context.Context, queryText string, nResults int) (*QueryResult, error) {
	params := HybridSearchParams{
		Query: map[string]interface{}{
			"query_text": queryText,
		},
		NResults: nResults,
		Rank: RankConfig{
			RRF: RRFConfig{K: 60},
		},
	}

	// If embedding function is set, generate embedding for text query
	if c.config.EmbeddingFunction != nil {
		queryEmb, err := c.config.EmbeddingFunction.EmbedQuery(ctx, queryText)
		if err == nil {
			params.KNN = map[string]interface{}{
				"query_embeddings": [][]float32{queryEmb},
			}
		}
	}

	return c.HybridSearch(ctx, params)
}