// Example: Collection management operations
package main

import (
	"context"
	"fmt"
	"log"

	seekdb "github.com/oceanbase/seekdb-go"
)

func main() {
	ctx := context.Background()

	// Create admin client
	admin, err := seekdb.NewAdminClient(seekdb.AdminConfig{
		Host: "127.0.0.1",
		Port: 2881,
		User: "root",
	})
	if err != nil {
		log.Fatalf("Failed to create admin client: %v", err)
	}
	defer admin.Close()

	// Create database
	dbName := "collection_demo"
	err = admin.CreateDatabase(ctx, dbName)
	if err != nil {
		log.Printf("Note: database may exist: %v", err)
	}

	// Create data client
	client, err := seekdb.NewClient(seekdb.ClientConfig{
		Host:     "127.0.0.1",
		Port:     2881,
		Database: dbName,
		User:     "root",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// List existing collections
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}
	fmt.Printf("Existing collections: %d\n", len(collections))

	// Create a collection
	collection, err := client.CreateCollection(ctx, "products", seekdb.CollectionConfig{
		Dimension:      64,
		DistanceMetric: seekdb.DistanceCosine,
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}
	fmt.Printf("Created collection: %s\n", collection.Name())

	// Check if collection exists
	exists, err := client.HasCollection(ctx, "products")
	if err != nil {
		log.Fatalf("Failed to check collection: %v", err)
	}
	fmt.Printf("Collection exists: %v\n", exists)

	// Fork the collection
	forked, err := collection.Fork(ctx, seekdb.ForkOptions{
		Name:     "products_backup",
	})
	if err != nil {
		log.Printf("Failed to fork: %v", err)
	} else {
		fmt.Printf("Forked collection: %s\n", forked.Name())
	}

	// Describe collection
	desc, err := collection.Describe(ctx)
	if err != nil {
		log.Fatalf("Failed to describe: %v", err)
	}
	fmt.Printf("\nCollection Description:\n")
	fmt.Printf("  Name: %s\n", desc.Name)
	fmt.Printf("  Dimension: %d\n", desc.Dimension)
	fmt.Printf("  Distance: %s\n", desc.DistanceMetric)
	fmt.Printf("  Documents: %d\n", desc.DocumentCount)

	// Count collections
	count, err := client.CountCollections(ctx)
	if err != nil {
		log.Fatalf("Failed to count: %v", err)
	}
	fmt.Printf("\nTotal collections: %d\n", count)

	// Cleanup
	fmt.Println("\nCleaning up...")
	client.DeleteCollection(ctx, "products")
	client.DeleteCollection(ctx, "products_backup")
	admin.DeleteDatabase(ctx, dbName)
	fmt.Println("Done!")
}