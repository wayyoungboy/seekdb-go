package seekdb

import (
	"testing"
)

func TestDatabaseInfo(t *testing.T) {
	info := DatabaseInfo{
		Name:      "testdb",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_unicode_ci",
		Tenant:    "test",
	}

	if info.Name != "testdb" {
		t.Errorf("Name = %q, want %q", info.Name, "testdb")
	}
	if info.Charset != "utf8mb4" {
		t.Errorf("Charset = %q, want %q", info.Charset, "utf8mb4")
	}
	if info.Collation != "utf8mb4_unicode_ci" {
		t.Errorf("Collation = %q, want %q", info.Collation, "utf8mb4_unicode_ci")
	}
	if info.Tenant != "test" {
		t.Errorf("Tenant = %q, want %q", info.Tenant, "test")
	}
}

func TestCollectionInfo(t *testing.T) {
	info := CollectionInfo{
		Name:              "test_collection",
		Dimension:         128,
		DistanceMetric:    DistanceCosine,
		EmbeddingFunction: "openai/text-embedding-3-small",
		Metadata:          map[string]string{"description": "test"},
	}

	if info.Name != "test_collection" {
		t.Errorf("Name = %q, want %q", info.Name, "test_collection")
	}
	if info.Dimension != 128 {
		t.Errorf("Dimension = %d, want %d", info.Dimension, 128)
	}
	if info.DistanceMetric != DistanceCosine {
		t.Errorf("DistanceMetric = %q, want %q", info.DistanceMetric, DistanceCosine)
	}
	if info.EmbeddingFunction != "openai/text-embedding-3-small" {
		t.Errorf("EmbeddingFunction = %q, want %q", info.EmbeddingFunction, "openai/text-embedding-3-small")
	}
	if info.Metadata["description"] != "test" {
		t.Errorf("Metadata[description] = %q, want %q", info.Metadata["description"], "test")
	}
}

func TestDocument(t *testing.T) {
	doc := Document{
		ID:        "doc1",
		Content:   "Hello world",
		Embedding: []float32{1.0, 2.0, 3.0},
		Metadata:  map[string]interface{}{"category": "test"},
	}

	if doc.ID != "doc1" {
		t.Errorf("ID = %q, want %q", doc.ID, "doc1")
	}
	if doc.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", doc.Content, "Hello world")
	}
	if len(doc.Embedding) != 3 {
		t.Errorf("Embedding length = %d, want %d", len(doc.Embedding), 3)
	}
	if doc.Metadata["category"] != "test" {
		t.Errorf("Metadata[category] = %q, want %q", doc.Metadata["category"], "test")
	}
}

func TestQueryResult(t *testing.T) {
	result := QueryResult{
		IDs:        [][]string{{"id1", "id2"}, {"id3", "id4"}},
		Documents:  [][]string{{"doc1", "doc2"}, {"doc3", "doc4"}},
		Embeddings: [][][]float32{{[]float32{1.0, 2.0}, []float32{3.0, 4.0}}, {[]float32{5.0, 6.0}, []float32{7.0, 8.0}}},
		Metadatas:  [][]map[string]interface{}{{map[string]interface{}{"k": "v1"}, map[string]interface{}{"k": "v2"}}},
		Distances:  [][]float32{{0.1, 0.2}, {0.3, 0.4}},
	}

	if len(result.IDs) != 2 {
		t.Errorf("IDs length = %d, want %d", len(result.IDs), 2)
	}
	if len(result.IDs[0]) != 2 {
		t.Errorf("IDs[0] length = %d, want %d", len(result.IDs[0]), 2)
	}
	if len(result.Documents) != 2 {
		t.Errorf("Documents length = %d, want %d", len(result.Documents), 2)
	}
	if len(result.Distances) != 2 {
		t.Errorf("Distances length = %d, want %d", len(result.Distances), 2)
	}
}

func TestGetResult(t *testing.T) {
	result := GetResult{
		IDs:        []string{"id1", "id2", "id3"},
		Documents:  []string{"doc1", "doc2", "doc3"},
		Embeddings: [][]float32{[]float32{1.0, 2.0}, []float32{3.0, 4.0}, []float32{5.0, 6.0}},
		Metadatas:  []map[string]interface{}{map[string]interface{}{"k": "v1"}, map[string]interface{}{"k": "v2"}},
	}

	if len(result.IDs) != 3 {
		t.Errorf("IDs length = %d, want %d", len(result.IDs), 3)
	}
	if len(result.Documents) != 3 {
		t.Errorf("Documents length = %d, want %d", len(result.Documents), 3)
	}
	if len(result.Embeddings) != 3 {
		t.Errorf("Embeddings length = %d, want %d", len(result.Embeddings), 3)
	}
}

func TestHybridSearchConfig(t *testing.T) {
	config := HybridSearchConfig{
		WhereDocument: map[string]interface{}{"$contains": "test"},
		Where:         map[string]interface{}{"category": "tech"},
		NResults:      10,
		QueryEmbeddings: [][]float32{[]float32{1.0, 2.0, 3.0}},
		KNNWhere:      map[string]interface{}{},
		KNNNResults:   5,
		Rank: RankConfig{
			RRF: RRFConfig{K: 60},
		},
	}

	if config.NResults != 10 {
		t.Errorf("NResults = %d, want %d", config.NResults, 10)
	}
	if config.Rank.RRF.K != 60 {
		t.Errorf("RRF.K = %d, want %d", config.Rank.RRF.K, 60)
	}
	if len(config.QueryEmbeddings) != 1 {
		t.Errorf("QueryEmbeddings length = %d, want %d", len(config.QueryEmbeddings), 1)
	}
}

func TestRankConfig(t *testing.T) {
	config := RankConfig{
		RRF: RRFConfig{K: 60},
	}

	if config.RRF.K != 60 {
		t.Errorf("RRF.K = %d, want %d", config.RRF.K, 60)
	}
}

func TestRRFConfig(t *testing.T) {
	config := RRFConfig{K: 60}

	if config.K != 60 {
		t.Errorf("K = %d, want %d", config.K, 60)
	}
}

func TestRRFConfigDefault(t *testing.T) {
	// Default K value should be 60 according to documentation
	config := RRFConfig{}

	// Note: the struct doesn't have a default value set
	// Users should use K: 60 for the default
	if config.K != 0 {
		t.Errorf("Unset K = %d, want 0 (user should set to 60)", config.K)
	}
}