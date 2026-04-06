// Example: Simple vector search with seekdb-go
package main

import (
	"context"
	"fmt"
	"log"

	seekdb "github.com/oceanbase/seekdb-go"
)

func main() {
	// Create admin client and database
	admin, err := seekdb.NewAdminClient(seekdb.AdminConfig{
		Host: "127.0.0.1",
		Port: 2881,
		User: "root",
		// Password can be set via SEEKDB_PASSWORD environment variable
	})
	if err != nil {
		log.Fatalf("Failed to create admin client: %v", err)
	}
	defer admin.Close()

	ctx := context.Background()

	// Create database
	err = admin.CreateDatabase(ctx, "vector_example")
	if err != nil {
		log.Printf("Note: database may already exist: %v", err)
	}

	// Create client for data operations
	client, err := seekdb.NewClient(seekdb.ClientConfig{
		Host:     "127.0.0.1",
		Port:     2881,
		Database: "vector_example",
		User:     "root",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create a collection with HNSW index
	collection, err := client.CreateCollection(ctx, "documents", seekdb.CollectionConfig{
		Dimension:      128,
		DistanceMetric: seekdb.DistanceCosine,
		VectorIndex:    seekdb.DefaultHNSWConfig(128, seekdb.DistanceCosine),
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	fmt.Printf("Created collection: %s\n", collection.Name())

	// Add documents with embeddings
	documents := []string{
		"Machine learning is a subset of artificial intelligence",
		"Python is a popular programming language",
		"Vector databases enable semantic search",
	}

	// Generate random embeddings for demo (in practice, use an embedding function)
	embeddings := generateRandomEmbeddings(128, len(documents))

	err = collection.Add(ctx, seekdb.AddParams{
		IDs:        []string{"doc1", "doc2", "doc3"},
		Documents:  documents,
		Embeddings: embeddings,
		Metadatas: []map[string]interface{}{
			{"category": "AI"},
			{"category": "Programming"},
			{"category": "Database"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Println("Added documents to collection")

	// Query for similar documents
	queryEmbedding := generateRandomEmbeddings(128, 1)[0]

	results, err := collection.Query(ctx, seekdb.QueryParams{
		QueryEmbeddings: [][]float32{queryEmbedding},
		NResults:        3,
	})
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Println("\nQuery results:")
	for i, id := range results.IDs[0] {
		fmt.Printf("  %d. ID: %s, Distance: %.4f\n", i+1, id, results.Distances[0][i])
		if len(results.Documents[0]) > i {
			fmt.Printf("     Document: %s\n", results.Documents[0][i])
		}
	}

	// Cleanup
	err = client.DeleteCollection(ctx, "documents")
	if err != nil {
		log.Printf("Failed to delete collection: %v", err)
	}

	err = admin.DeleteDatabase(ctx, "vector_example")
	if err != nil {
		log.Printf("Failed to delete database: %v", err)
	}

	fmt.Println("\nCleanup complete")
}

func generateRandomEmbeddings(dimension, count int) [][]float32 {
	embeddings := make([][]float32, count)
	for i := 0; i < count; i++ {
		embeddings[i] = make([]float32, dimension)
		// In practice, use an embedding function to generate real embeddings
		// This is just a placeholder
		for j := 0; j < dimension; j++ {
			embeddings[i][j] = float32(j%10) / 10.0
		}
	}
	return embeddings
}
