package seekdb

// Common types used across the SDK.

// DatabaseInfo represents metadata about a database.
type DatabaseInfo struct {
	Name      string
	Charset   string
	Collation string
	Tenant    string // Only for OceanBase Database
}

// CollectionInfo represents metadata about a collection.
type CollectionInfo struct {
	Name              string
	Dimension         int
	DistanceMetric    DistanceMetric
	EmbeddingFunction string
	Metadata          map[string]string
}

// Document represents a single document in a collection.
type Document struct {
	ID        string
	Content   string
	Embedding []float32
	Metadata  map[string]interface{}
}

// QueryResult holds the results of a vector similarity query.
type QueryResult struct {
	IDs        [][]string                 // IDs for each query result set
	Documents  [][]string                 // Documents for each query result set
	Embeddings [][][]float32              // Embeddings for each query result set
	Metadatas  [][]map[string]interface{} // Metadatas for each query result set
	Distances  [][]float32                // Distances for each query result set
}

// GetResult holds the results of a get operation.
type GetResult struct {
	IDs        []string
	Documents  []string
	Embeddings [][]float32
	Metadatas  []map[string]interface{}
}

// HybridSearchConfig holds the configuration for hybrid search.
type HybridSearchConfig struct {
	// Full-text search query
	WhereDocument map[string]interface{}
	Where         map[string]interface{}
	NResults      int

	// Vector search query
	QueryEmbeddings [][]float32
	KNNWhere        map[string]interface{}
	KNNNResults     int

	// Ranking configuration
	Rank RankConfig
}

// RankConfig holds the ranking configuration for hybrid search.
type RankConfig struct {
	RRF RRFConfig // Reciprocal Rank Fusion
}

// RRFConfig holds RRF (Reciprocal Rank Fusion) parameters.
type RRFConfig struct {
	K int // RRF parameter (default: 60)
}

// IncludeOptions controls which fields are included in query results.
type IncludeOptions struct {
	Documents  bool // Include document content
	Embeddings bool // Include embedding vectors
	Metadatas  bool // Include metadata
	Distances  bool // Include distance scores (only for vector queries)
}

// DefaultInclude returns the default include options for queries.
func DefaultInclude() IncludeOptions {
	return IncludeOptions{
		Documents:  true,
		Embeddings: false,
		Metadatas:  true,
		Distances:  true,
	}
}

// IncludeAll returns include options with all fields enabled.
func IncludeAll() IncludeOptions {
	return IncludeOptions{
		Documents:  true,
		Embeddings: true,
		Metadatas:  true,
		Distances:  true,
	}
}

// IncludeNone returns include options with no fields enabled.
func IncludeNone() IncludeOptions {
	return IncludeOptions{}
}
