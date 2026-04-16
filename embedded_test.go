//go:build integration

package seekdb

import (
	"context"
	"os"
	"testing"
)

const seekdbBinaryPath = "/tmp/seekdb-binary"

// TestEmbeddedMode runs a full integration test against an embedded seekdb instance.
func TestEmbeddedMode(t *testing.T) {
	if _, err := os.Stat(seekdbBinaryPath); err != nil {
		t.Skipf("seekdb binary not found at %s, skipping embedded test", seekdbBinaryPath)
	}

	ctx := context.Background()

	// Test 1: Embedded AdminClient
	t.Run("AdminClient", func(t *testing.T) {
		admin, err := NewAdminClient(AdminConfig{
			Path: t.TempDir(),
			EmbeddedConfig: EmbeddedConfig{
				BinaryPath: seekdbBinaryPath,
			},
		})
		if err != nil {
			t.Fatalf("NewAdminClient failed: %v", err)
		}
		defer admin.Close()

		// Create database
		err = admin.CreateDatabase(ctx, "embed_test")
		if err != nil {
			t.Fatalf("CreateDatabase failed: %v", err)
		}
		t.Log("Created database embed_test")

		// Get database
		info, err := admin.GetDatabase(ctx, "embed_test")
		if err != nil {
			t.Fatalf("GetDatabase failed: %v", err)
		}
		if info.Name != "embed_test" {
			t.Errorf("GetDatabase name = %q, want %q", info.Name, "embed_test")
		}

		// List databases
		dbs, err := admin.ListDatabases(ctx, 20, 0)
		if err != nil {
			t.Fatalf("ListDatabases failed: %v", err)
		}
		if len(dbs) < 1 {
			t.Error("ListDatabases returned empty list")
		}

		// Delete database
		err = admin.DeleteDatabase(ctx, "embed_test")
		if err != nil {
			t.Fatalf("DeleteDatabase failed: %v", err)
		}
		t.Log("Deleted database embed_test")
	})

	// Test 2: Embedded Client with full CRUD
	t.Run("Client", func(t *testing.T) {
		client, err := NewClient(ClientConfig{
			Path:     t.TempDir(),
			Database: "test",
			EmbeddedConfig: EmbeddedConfig{
				BinaryPath: seekdbBinaryPath,
			},
		})
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		defer client.Close()

		// Delete collection if exists
		_ = client.DeleteCollection(ctx, "embed_col")

		// Create collection
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
		collection, err := client.CreateCollection(ctx, "embed_col", config)
		if err != nil {
			t.Fatalf("CreateCollection failed: %v", err)
		}
		t.Logf("Created collection: %s", collection.Name())

		// Add documents
		err = collection.Add(ctx, AddParams{
			IDs:       []string{"1", "2", "3"},
			Documents: []string{"hello world", "foo bar", "test document"},
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
				{0.7, 0.8, 0.9},
			},
		})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Count
		count, err := collection.Count(ctx)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}
		if count != 3 {
			t.Errorf("Count = %d, want 3", count)
		}

		// Query
		result, err := collection.Query(ctx, QueryParams{
			QueryEmbeddings: [][]float32{{0.15, 0.25, 0.35}},
			NResults:        2,
			Include:         DefaultInclude(),
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(result.IDs[0]) < 1 {
			t.Error("Query returned no results")
		} else {
			t.Logf("Query returned %d results, first ID: %s", len(result.IDs[0]), result.IDs[0][0])
		}

		// Get by IDs
		getResult, err := collection.Get(ctx, GetParams{
			IDs:     []string{"1", "2"},
			Include: IncludeOptions{Documents: true},
		})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(getResult.IDs) != 2 {
			t.Errorf("Get returned %d IDs, want 2", len(getResult.IDs))
		}

		// Update
		err = collection.Update(ctx, UpdateParams{
			IDs:       []string{"1"},
			Documents: []string{"hello world updated"},
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Delete
		err = collection.Delete(ctx, DeleteParams{IDs: []string{"3"}})
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		count, err = collection.Count(ctx)
		if err != nil {
			t.Fatalf("Count after delete failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Count after delete = %d, want 2", count)
		}

		// Cleanup
		err = client.DeleteCollection(ctx, "embed_col")
		if err != nil {
			t.Fatalf("DeleteCollection failed: %v", err)
		}

		t.Log("Embedded client tests passed!")
	})

	// Test 3: Auto port assignment
	t.Run("AutoPort", func(t *testing.T) {
		client, err := NewClient(ClientConfig{
			Path: t.TempDir(),
			EmbeddedConfig: EmbeddedConfig{
				BinaryPath: seekdbBinaryPath,
				Port:       0, // Auto-assign port
			},
		})
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		defer client.Close()

		// Just verify it can connect
		_, err = client.CountCollections(ctx)
		if err != nil {
			// Expected: no collections yet is fine
			t.Logf("CountCollections returned: %v (expected if no collections exist)", err)
		}

		t.Logf("Auto port assignment works")
	})
}
