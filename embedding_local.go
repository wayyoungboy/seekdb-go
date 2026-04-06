package seekdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HuggingFaceInferenceResponse represents the response from HuggingFace Inference API.
type HuggingFaceInferenceResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// EmbedDocuments generates embeddings using HuggingFace Inference API.
func (f *HuggingFaceEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	// HuggingFace Inference API
	url := fmt.Sprintf("https://api-inference.huggingface.co/pipeline/feature-extraction/%s", f.config.Model)

	jsonBody, err := json.Marshal(documents)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.config.APIKey))

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HuggingFace API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embeddings [][]float32
	if err := json.Unmarshal(body, &embeddings); err != nil {
		// Try parsing as single array of float32
		var singleEmb []float32
		if err := json.Unmarshal(body, &singleEmb); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		embeddings = [][]float32{singleEmb}
	}

	return embeddings, nil
}

func (f *HuggingFaceEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

// LocalEmbeddingFunction represents a local embedding model.
// This can be used with local ONNX models or other local implementations.
type LocalEmbeddingFunction struct {
	modelPath  string
	dimension  int
	embedFunc  func(ctx context.Context, texts []string) ([][]float32, error)
}

// NewLocalEmbeddingFunction creates a local embedding function with a custom implementation.
func NewLocalEmbeddingFunction(dimension int, embedFunc func(ctx context.Context, texts []string) ([][]float32, error)) *LocalEmbeddingFunction {
	return &LocalEmbeddingFunction{
		dimension: dimension,
		embedFunc: embedFunc,
	}
}

func (f *LocalEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if f.embedFunc == nil {
		return nil, fmt.Errorf("no embedding function provided")
	}
	return f.embedFunc(ctx, documents)
}

func (f *LocalEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

func (f *LocalEmbeddingFunction) Name() string {
	return "local"
}

func (f *LocalEmbeddingFunction) Dimension() int {
	return f.dimension
}

// MockEmbeddingFunction is a simple mock for testing purposes.
type MockEmbeddingFunctionSimple struct {
	dimension int
}

// NewMockEmbedding creates a mock embedding function for testing.
func NewMockEmbedding(dimension int) *MockEmbeddingFunctionSimple {
	return &MockEmbeddingFunctionSimple{dimension: dimension}
}

func (f *MockEmbeddingFunctionSimple) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	embeddings := make([][]float32, len(documents))
	for i := range embeddings {
		embeddings[i] = make([]float32, f.dimension)
		// Generate deterministic embeddings based on document hash
		for j := range embeddings[i] {
			embeddings[i][j] = float32((i + j) % 100) / 100.0
		}
	}
	return embeddings, nil
}

func (f *MockEmbeddingFunctionSimple) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	emb := make([]float32, f.dimension)
	for i := range emb {
		emb[i] = float32(i % 100) / 100.0
	}
	return emb, nil
}

func (f *MockEmbeddingFunctionSimple) Name() string {
	return "mock"
}

func (f *MockEmbeddingFunctionSimple) Dimension() int {
	return f.dimension
}

// EmbeddingFunctionRegistry maintains a registry of embedding functions.
type EmbeddingFunctionRegistry struct {
	functions map[string]EmbeddingFunction
}

// NewEmbeddingFunctionRegistry creates a new registry.
func NewEmbeddingFunctionRegistry() *EmbeddingFunctionRegistry {
	return &EmbeddingFunctionRegistry{
		functions: make(map[string]EmbeddingFunction),
	}
}

// Register adds an embedding function to the registry.
func (r *EmbeddingFunctionRegistry) Register(name string, fn EmbeddingFunction) {
	r.functions[name] = fn
}

// Get retrieves an embedding function by name.
func (r *EmbeddingFunctionRegistry) Get(name string) (EmbeddingFunction, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

// List returns all registered embedding function names.
func (r *EmbeddingFunctionRegistry) List() []string {
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	return names
}