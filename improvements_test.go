package seekdb

import (
	"testing"
	"time"
)

func TestIncludeOptions(t *testing.T) {
	// Test DefaultInclude
	defaultInc := DefaultInclude()
	if !defaultInc.Documents {
		t.Error("DefaultInclude should include Documents")
	}
	if !defaultInc.Metadatas {
		t.Error("DefaultInclude should include Metadatas")
	}
	if !defaultInc.Distances {
		t.Error("DefaultInclude should include Distances")
	}
	if defaultInc.Embeddings {
		t.Error("DefaultInclude should not include Embeddings")
	}

	// Test IncludeAll
	allInc := IncludeAll()
	if !allInc.Documents || !allInc.Embeddings || !allInc.Metadatas || !allInc.Distances {
		t.Error("IncludeAll should include all fields")
	}

	// Test IncludeNone
	noneInc := IncludeNone()
	if noneInc.Documents || noneInc.Embeddings || noneInc.Metadatas || noneInc.Distances {
		t.Error("IncludeNone should not include any fields")
	}
}

func TestConnectionPoolConfig(t *testing.T) {
	config := DefaultConnectionPoolConfig()
	if config.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want 25", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want 5", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 5m", config.ConnMaxLifetime)
	}
}

func TestCollectionTableName(t *testing.T) {
	tests := []struct {
		name      string
		wantTable string
	}{
		{"my_collection", "c$v1$my_collection"},
		{"test", "c$v1$test"},
		{"", "c$v1$"},
	}

	for _, tt := range tests {
		result := collectionTableName(tt.name)
		if result != tt.wantTable {
			t.Errorf("collectionTableName(%q) = %q, want %q", tt.name, result, tt.wantTable)
		}
	}
}

func TestParseCollectionName(t *testing.T) {
	tests := []struct {
		tableName string
		wantName  string
	}{
		{"c$v1$my_collection", "my_collection"},
		{"c$v1$test", "test"},
		{"plain_name", "plain_name"},
		{"", ""},
	}

	for _, tt := range tests {
		result := parseCollectionName(tt.tableName)
		if result != tt.wantName {
			t.Errorf("parseCollectionName(%q) = %q, want %q", tt.tableName, result, tt.wantName)
		}
	}
}

func TestBuildSelectFields(t *testing.T) {
	collection := &Collection{name: "test"}

	tests := []struct {
		name        string
		include     IncludeOptions
		withDistance bool
		wantFields  []string
	}{
		{
			name:         "default",
			include:      DefaultInclude(),
			withDistance: true,
			wantFields:   []string{"id", "document", "metadata", "VECTOR_DISTANCE(embedding, ?) as distance"},
		},
		{
			name:         "all",
			include:      IncludeAll(),
			withDistance: true,
			wantFields:   []string{"id", "document", "embedding", "metadata", "VECTOR_DISTANCE(embedding, ?) as distance"},
		},
		{
			name:         "none",
			include:      IncludeNone(),
			withDistance: false,
			wantFields:   []string{"id"},
		},
		{
			name:         "only documents",
			include:      IncludeOptions{Documents: true},
			withDistance: false,
			wantFields:   []string{"id", "document"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collection.buildSelectFields(tt.include, tt.withDistance)
			for _, field := range tt.wantFields {
				if !containsString(result, field) {
					t.Errorf("buildSelectFields missing field %q in result %q", field, result)
				}
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := ErrCollectionNotFound
	wrapped := WrapError(originalErr, "failed to get collection")

	if wrapped == nil {
		t.Error("WrapError should not return nil")
	}

	wrappedStr := wrapped.Error()
	if !containsString(wrappedStr, "failed to get collection") {
		t.Errorf("WrapError missing message in %q", wrappedStr)
	}
	if !containsString(wrappedStr, "collection not found") {
		t.Errorf("WrapError missing original error in %q", wrappedStr)
	}
}

func TestIsNotFoundError(t *testing.T) {
	if !IsNotFoundError(ErrDatabaseNotFound) {
		t.Error("ErrDatabaseNotFound should be a not found error")
	}
	if !IsNotFoundError(ErrCollectionNotFound) {
		t.Error("ErrCollectionNotFound should be a not found error")
	}
	if IsNotFoundError(ErrInvalidConfig) {
		t.Error("ErrInvalidConfig should not be a not found error")
	}
}

func TestIsConnectionError(t *testing.T) {
	if !IsConnectionError(ErrNotConnected) {
		t.Error("ErrNotConnected should be a connection error")
	}
	if !IsConnectionError(ErrConnectionFailed) {
		t.Error("ErrConnectionFailed should be a connection error")
	}
	if IsConnectionError(ErrCollectionNotFound) {
		t.Error("ErrCollectionNotFound should not be a connection error")
	}
}

func TestIsValidationError(t *testing.T) {
	if !IsValidationError(ErrDatabaseNameEmpty) {
		t.Error("ErrDatabaseNameEmpty should be a validation error")
	}
	if !IsValidationError(ErrInvalidDimension) {
		t.Error("ErrInvalidDimension should be a validation error")
	}
	if IsValidationError(ErrCollectionNotFound) {
		t.Error("ErrCollectionNotFound should not be a validation error")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}