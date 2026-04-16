//go:build integration

package seekdb

import (
	"context"
	"testing"
	"time"
)

// TestServerModeIntegration runs a full integration test against a running seekdb instance.
// Requires: docker run -d --name seekdb-test -p 2881:2881 oceanbase/seekdb:latest
func TestServerModeIntegration(t *testing.T) {
	ctx := context.Background()
	cfg := ClientConfig{
		Host:     "127.0.0.1",
		Port:     2881,
		Database: "test",
		User:     "root",
		Password: "",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		err := client.ensureConnection()
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if !client.connected {
		t.Fatal("Failed to connect to seekdb server after retries")
	}

	// Test 1: Delete collection if exists from previous run
	_ = client.DeleteCollection(ctx, "go_test")

	// Test 2: Create collection
	config := CollectionConfig{
		Dimension:      3,
		DistanceMetric: DistanceCosine,
		VectorIndex: &HNSWConfig{
			Dimension:      3,
			DistanceMetric: DistanceCosine,
			M:              16,
			EfConstruction: 128,
		},
	}
	collection, err := client.CreateCollection(ctx, "go_test", config)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	t.Logf("Created collection: %s", collection.Name())

	// Test 3: HasCollection
	exists, err := client.HasCollection(ctx, "go_test")
	if err != nil {
		t.Fatalf("HasCollection failed: %v", err)
	}
	if !exists {
		t.Fatal("HasCollection should return true for existing collection")
	}

	// Test 4: GetCollection
	got, err := client.GetCollection(ctx, "go_test")
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}
	if got.Name() != "go_test" {
		t.Errorf("GetCollection name = %q, want %q", got.Name(), "go_test")
	}

	// Test 5: CountCollections
	count, err := client.CountCollections(ctx)
	if err != nil {
		t.Fatalf("CountCollections failed: %v", err)
	}
	if count < 1 {
		t.Errorf("CountCollections = %d, want >= 1", count)
	}

	// Test 6: Add documents
	err = collection.Add(ctx, AddParams{
		IDs:       []string{"1", "2", "3"},
		Documents: []string{"hello world", "foo bar", "test document"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
			{0.7, 0.8, 0.9},
		},
		Metadatas: []map[string]interface{}{
			{"source": "test"},
			{"source": "prod"},
			{"source": "dev"},
		},
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	t.Log("Added 3 documents")

	// Test 7: Count documents
	docCount, err := collection.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if docCount != 3 {
		t.Errorf("Count = %d, want 3", docCount)
	}

	// Test 8: Get documents by IDs
	result, err := collection.Get(ctx, GetParams{
		IDs: []string{"1", "2"},
		Include: IncludeOptions{
			Documents: true,
			Metadatas: true,
		},
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result.IDs) != 2 {
		t.Errorf("Get returned %d IDs, want 2", len(result.IDs))
	}

	// Test 9: Query (vector similarity search)
	queryResult, err := collection.Query(ctx, QueryParams{
		QueryEmbeddings: [][]float32{{0.15, 0.25, 0.35}},
		NResults:        2,
		Include:         DefaultInclude(),
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(queryResult.IDs[0]) < 1 {
		t.Error("Query returned no results")
	} else {
		t.Logf("Query returned %d results, first ID: %s, distance: %.4f",
			len(queryResult.IDs[0]), queryResult.IDs[0][0], queryResult.Distances[0][0])
	}

	// Test 10: Peek
	peekResult, err := collection.Peek(ctx, 2)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if len(peekResult.IDs) > 2 {
		t.Errorf("Peek returned %d docs, want <= 2", len(peekResult.IDs))
	}

	// Test 11: Update documents
	err = collection.Update(ctx, UpdateParams{
		IDs:       []string{"1"},
		Documents: []string{"hello world updated"},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	t.Log("Updated document 1")

	// Test 12: Upsert documents
	err = collection.Upsert(ctx, UpsertParams{
		IDs:       []string{"1", "4"},
		Documents: []string{"hello world upserted", "new document"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.5, 0.5, 0.5},
		},
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	t.Log("Upserted documents")

	// Test 13: Count after upsert
	docCount, err = collection.Count(ctx)
	if err != nil {
		t.Fatalf("Count after upsert failed: %v", err)
	}
	if docCount != 4 {
		t.Errorf("Count after upsert = %d, want 4", docCount)
	}

	// Test 14: Delete by IDs
	err = collection.Delete(ctx, DeleteParams{IDs: []string{"4"}})
	if err != nil {
		t.Fatalf("Delete by ID failed: %v", err)
	}
	docCount, err = collection.Count(ctx)
	if err != nil {
		t.Fatalf("Count after delete failed: %v", err)
	}
	if docCount != 3 {
		t.Errorf("Count after delete = %d, want 3", docCount)
	}

	// Test 15: ListCollections
	collections, err := client.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	found := false
	for _, c := range collections {
		if c.Name == "go_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListCollections did not find go_test collection")
	}

	// Test 16: Fork collection
	_ = client.DeleteCollection(ctx, "go_test_fork")
	forked, err := collection.Fork(ctx, ForkOptions{Name: "go_test_fork"})
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}
	t.Logf("Forked collection: %s", forked.Name())

	// Test 17: Describe collection
	desc, err := collection.Describe(ctx)
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}
	if desc.Name != "go_test" {
		t.Errorf("Describe name = %q, want %q", desc.Name, "go_test")
	}

	// Test 18: DeleteCollection
	err = client.DeleteCollection(ctx, "go_test")
	if err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}
	err = client.DeleteCollection(ctx, "go_test_fork")
	if err != nil {
		t.Fatalf("DeleteCollection fork failed: %v", err)
	}

	// Verify deletion
	exists, err = client.HasCollection(ctx, "go_test")
	if err != nil {
		t.Fatalf("HasCollection after delete failed: %v", err)
	}
	if exists {
		t.Fatal("HasCollection should return false after deletion")
	}

	t.Log("All integration tests passed!")
}

// TestAdminClientServerMode tests AdminClient against a running seekdb server.
func TestAdminClientServerMode(t *testing.T) {
	ctx := context.Background()
	cfg := AdminConfig{
		Host: "127.0.0.1",
		Port: 2881,
		User: "root",
	}

	admin, err := NewAdminClient(cfg)
	if err != nil {
		t.Fatalf("NewAdminClient failed: %v", err)
	}
	defer admin.Close()

	// Test CreateDatabase
	_ = admin.DeleteDatabase(ctx, "go_test_db")

	err = admin.CreateDatabase(ctx, "go_test_db")
	if err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}
	t.Log("Created database go_test_db")

	// Test GetDatabase
	info, err := admin.GetDatabase(ctx, "go_test_db")
	if err != nil {
		t.Fatalf("GetDatabase failed: %v", err)
	}
	if info.Name != "go_test_db" {
		t.Errorf("GetDatabase name = %q, want %q", info.Name, "go_test_db")
	}

	// Test ListDatabases
	dbs, err := admin.ListDatabases(ctx, 20, 0)
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}
	if len(dbs) < 1 {
		t.Error("ListDatabases returned empty list")
	}

	// Test DeleteDatabase
	err = admin.DeleteDatabase(ctx, "go_test_db")
	if err != nil {
		t.Fatalf("DeleteDatabase failed: %v", err)
	}
	t.Log("Deleted database go_test_db")

	t.Log("AdminClient integration tests passed!")
}
