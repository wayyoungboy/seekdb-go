package seekdb

import "errors"

// Common errors returned by the seekdb SDK.
var (
	// Connection errors
	ErrNotConnected         = errors.New("not connected to seekdb")
	ErrConnectionFailed     = errors.New("failed to connect to seekdb")
	ErrEmbeddedNotSupported = errors.New("embedded mode not supported on this platform")

	// Database errors
	ErrDatabaseNotFound  = errors.New("database not found")
	ErrDatabaseExists    = errors.New("database already exists")
	ErrDatabaseNameEmpty = errors.New("database name cannot be empty")

	// Collection errors
	ErrCollectionNotFound  = errors.New("collection not found")
	ErrCollectionExists    = errors.New("collection already exists")
	ErrCollectionNameEmpty = errors.New("collection name cannot be empty")
	ErrInvalidDimension    = errors.New("invalid vector dimension")

	// Data errors
	ErrIDRequired        = errors.New("id is required")
	ErrEmbeddingRequired = errors.New("embedding is required when no embedding function is set")
	ErrInvalidEmbedding  = errors.New("embedding dimension mismatch")

	// Query errors
	ErrInvalidFilter          = errors.New("invalid filter expression")
	ErrQueryEmbeddingRequired = errors.New("query embedding is required")

	// Configuration errors
	ErrInvalidConfig = errors.New("invalid configuration")
)

// SeekdbError represents a detailed error from seekdb.
type SeekdbError struct {
	Code    int
	Message string
	Details string
}

func (e *SeekdbError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// NewSeekdbError creates a new SeekdbError.
func NewSeekdbError(code int, message, details string) *SeekdbError {
	return &SeekdbError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
