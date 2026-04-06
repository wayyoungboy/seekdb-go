package seekdb

import (
	"errors"
	"testing"
)

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		want  string
	}{
		{"ErrNotConnected", ErrNotConnected, "not connected to seekdb"},
		{"ErrConnectionFailed", ErrConnectionFailed, "failed to connect to seekdb"},
		{"ErrEmbeddedNotSupported", ErrEmbeddedNotSupported, "embedded mode not supported on this platform"},
		{"ErrDatabaseNotFound", ErrDatabaseNotFound, "database not found"},
		{"ErrDatabaseExists", ErrDatabaseExists, "database already exists"},
		{"ErrDatabaseNameEmpty", ErrDatabaseNameEmpty, "database name cannot be empty"},
		{"ErrCollectionNotFound", ErrCollectionNotFound, "collection not found"},
		{"ErrCollectionExists", ErrCollectionExists, "collection already exists"},
		{"ErrCollectionNameEmpty", ErrCollectionNameEmpty, "collection name cannot be empty"},
		{"ErrInvalidDimension", ErrInvalidDimension, "invalid vector dimension"},
		{"ErrIDRequired", ErrIDRequired, "id is required"},
		{"ErrEmbeddingRequired", ErrEmbeddingRequired, "embedding is required when no embedding function is set"},
		{"ErrInvalidEmbedding", ErrInvalidEmbedding, "embedding dimension mismatch"},
		{"ErrInvalidFilter", ErrInvalidFilter, "invalid filter expression"},
		{"ErrQueryEmbeddingRequired", ErrQueryEmbeddingRequired, "query embedding is required"},
		{"ErrInvalidConfig", ErrInvalidConfig, "invalid configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

func TestSeekdbError(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
		details string
		want    string
	}{
		{
			name:    "with details",
			code:    1001,
			message: "connection error",
			details: "timeout after 30s",
			want:    "connection error: timeout after 30s",
		},
		{
			name:    "without details",
			code:    1002,
			message: "query error",
			details: "",
			want:    "query error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewSeekdbError(tt.code, tt.message, tt.details)
			if err.Code != tt.code {
				t.Errorf("Code = %d, want %d", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}
			if err.Details != tt.details {
				t.Errorf("Details = %q, want %q", err.Details, tt.details)
			}
			if err.Error() != tt.want {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSeekdbErrorIs(t *testing.T) {
	err := NewSeekdbError(1001, "test error", "details")
	if errors.Is(err, ErrNotConnected) {
		t.Error("SeekdbError should not match ErrNotConnected via errors.Is")
	}

	// Test that SeekdbError is itself
	if !errors.Is(err, err) {
		t.Error("SeekdbError should match itself via errors.Is")
	}
}