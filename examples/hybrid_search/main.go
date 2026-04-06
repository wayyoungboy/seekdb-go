// Example: Hybrid search combining vector and full-text search
package main

import (
	"context"
	"fmt"
	"log"

	seekdb "github.com/oceanbase/seekdb-go"
)

func main() {
	ctx := context.Background()

	client, err := seekdb.NewClient(seekdb.ClientConfig{
		Host:     "127.0.0.1",
		Port:     2881,
		Database: "hybrid_demo",
		User:     "root",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create collection
	collection, err := client.CreateCollection(ctx, "articles", seekdb.CollectionConfig{
		Dimension:      128,
		DistanceMetric: seekdb.DistanceCosine,
		VectorIndex:    seekdb.DefaultHNSWConfig(128, seekdb.DistanceCosine),
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	// Create full-text index for hybrid search
	err = collection.CreateFulltextIndex(ctx)
	if err != nil {
		log.Printf("Note: fulltext index creation: %v", err)
	}

	// Add documents
	documents := []string{
		"Machine learning is transforming healthcare diagnostics",
		"Deep neural networks achieve state-of-the-art results",
		"Natural language processing enables chatbots",
		"Computer vision applications in autonomous driving",
		"Reinforcement learning for game playing AI",
	}

	// Generate sample embeddings (in practice, use real embeddings)
	embeddings := generateSampleEmbeddings(128, len(documents))

	err = collection.Add(ctx, seekdb.AddParams{
		IDs:        []string{"art1", "art2", "art3", "art4", "art5"},
		Documents:  documents,
		Embeddings: embeddings,
		Metadatas: []map[string]interface{}{
			{"field": "healthcare"},
			{"field": "research"},
			{"field": "nlp"},
			{"field": "vision"},
			{"field": "games"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	// Perform hybrid search
	queryEmb := generateSampleEmbeddings(128, 1)[0]

	results, err := collection.HybridSearch(ctx, seekdb.HybridSearchParams{
		Query: map[string]interface{}{
			"query_text": "AI and machine learning",
		},
		KNN: map[string]interface{}{
			"query_embeddings": [][]float32{queryEmb},
		},
		Rank: seekdb.RankConfig{
			RRF: seekdb.RRFConfig{K: 60},
		},
		NResults: 3,
	})
	if err != nil {
		log.Fatalf("Failed to perform hybrid search: %v", err)
	}

	fmt.Println("Hybrid Search Results:")
	for i, id := range results.IDs[0] {
		fmt.Printf("%d. ID: %s, Distance: %.4f\n", i+1, id, results.Distances[0][i])
		if len(results.Documents[0]) > i {
			fmt.Printf("   Document: %s\n", results.Documents[0][i])
		}
	}

	// Cleanup
	client.DeleteCollection(ctx, "articles")
}

func generateSampleEmbeddings(dimension, count int) [][]float32 {
	embeddings := make([][]float32, count)
	for i := 0; i < count; i++ {
		embeddings[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			embeddings[i][j] = float32((i*dimension+j)%100) / 100.0
		}
	}
	return embeddings
}