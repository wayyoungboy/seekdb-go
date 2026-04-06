// Example: Using OpenAI embeddings with seekdb-go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	seekdb "github.com/oceanbase/seekdb-go"
)

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create client
	client, err := seekdb.NewClient(seekdb.ClientConfig{
		Host:     "127.0.0.1",
		Port:     2881,
		Database: "embeddings_demo",
		User:     "root",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create embedding function
	embedFn := seekdb.NewOpenAIEmbeddingFunction("text-embedding-3-small", apiKey)

	// Create collection with embedding function
	collection, err := client.GetOrCreateCollection(ctx, "documents", seekdb.CollectionConfig{
		Dimension:         1536,
		DistanceMetric:    seekdb.DistanceCosine,
		EmbeddingFunction: embedFn,
		VectorIndex:       seekdb.DefaultHNSWConfig(1536, seekdb.DistanceCosine),
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	fmt.Printf("Created collection: %s\n", collection.Name())

	// Add documents (embeddings will be auto-generated)
	documents := []string{
		"Go is a programming language designed for simplicity and reliability",
		"Vector databases enable semantic search capabilities",
		"Machine learning models can understand text semantics",
		"OceanBase is a distributed relational database",
	}

	err = collection.Add(ctx, seekdb.AddParams{
		IDs:       []string{"doc1", "doc2", "doc3", "doc4"},
		Documents: documents,
		Metadatas: []map[string]interface{}{
			{"topic": "programming"},
			{"topic": "database"},
			{"topic": "ai"},
			{"topic": "database"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Println("Added documents to collection")

	// Query with text (will be auto-embedded)
	queryText := "database for AI applications"

	// Generate embedding for query
	queryEmb, err := embedFn.EmbedQuery(ctx, queryText)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}

	results, err := collection.Query(ctx, seekdb.QueryParams{
		QueryEmbeddings: [][]float32{queryEmb},
		NResults:        3,
		Where:           map[string]interface{}{"topic": "database"},
	})
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("\nQuery: %s\n\n", queryText)
	fmt.Println("Results:")
	for i, id := range results.IDs[0] {
		fmt.Printf("%d. ID: %s, Distance: %.4f\n", i+1, id, results.Distances[0][i])
		if len(results.Documents[0]) > i {
			fmt.Printf("   Document: %s\n", results.Documents[0][i])
		}
	}
}