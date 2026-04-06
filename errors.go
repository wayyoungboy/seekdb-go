package seekdb

import (
	"errors"
	"fmt"
)

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
	Cause   error // Underlying error
}

func (e *SeekdbError) Error() string {
	if e.Details != "" && e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Message, e.Details, e.Cause)
	}
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s (caused by: %v)", e.Message, e.Cause)
	}
	return e.Message
}

func (e *SeekdbError) Unwrap() error {
	return e.Cause
}

// NewSeekdbError creates a new SeekdbError.
func NewSeekdbError(code int, message, details string) *SeekdbError {
	return &SeekdbError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error with additional context.
func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// WrapErrorf wraps an existing error with formatted context.
func WrapErrorf(err error, format string, args ...interface{}) error {
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// IsNotFoundError checks if the error is a "not found" error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrDatabaseNotFound) ||
		errors.Is(err, ErrCollectionNotFound)
}

// IsConnectionError checks if the error is a connection error.
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrNotConnected) ||
		errors.Is(err, ErrConnectionFailed)
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrDatabaseNameEmpty) ||
		errors.Is(err, ErrCollectionNameEmpty) ||
		errors.Is(err, ErrInvalidDimension) ||
		errors.Is(err, ErrInvalidConfig) ||
		errors.Is(err, ErrInvalidEmbedding)
}
