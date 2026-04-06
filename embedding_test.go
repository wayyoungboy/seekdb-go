package seekdb

import (
	"testing"
)

func TestOpenAIEmbeddingFunction(t *testing.T) {
	fn := NewOpenAIEmbeddingFunction("text-embedding-3-small", "test-key")

	if fn.Name() != "openai/text-embedding-3-small" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "openai/text-embedding-3-small")
	}
	if fn.Dimension() != 1536 {
		t.Errorf("Dimension() = %d, want %d", fn.Dimension(), 1536)
	}
}

func TestOpenAIEmbeddingFunctionLarge(t *testing.T) {
	fn := NewOpenAIEmbeddingFunction("text-embedding-3-large", "test-key")

	if fn.Name() != "openai/text-embedding-3-large" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "openai/text-embedding-3-large")
	}
	if fn.Dimension() != 3072 {
		t.Errorf("Dimension() = %d, want %d", fn.Dimension(), 3072)
	}
}

func TestHuggingFaceEmbeddingFunction(t *testing.T) {
	fn := NewHuggingFaceEmbeddingFunction("sentence-transformers/all-MiniLM-L6-v2", "test-key")

	if fn.Name() != "huggingface/sentence-transformers/all-MiniLM-L6-v2" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "huggingface/sentence-transformers/all-MiniLM-L6-v2")
	}
	if fn.Dimension() != 768 {
		t.Errorf("Dimension() = %d, want %d", fn.Dimension(), 768)
	}
}

func TestEmbeddingFunctionConfig(t *testing.T) {
	config := EmbeddingFunctionConfig{
		Provider:   "openai",
		Model:      "text-embedding-3-small",
		APIKey:     "test-key",
		Parameters: map[string]interface{}{"batch_size": 100},
	}

	if config.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", config.Provider, "openai")
	}
	if config.Model != "text-embedding-3-small" {
		t.Errorf("Model = %q, want %q", config.Model, "text-embedding-3-small")
	}
	if config.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", config.APIKey, "test-key")
	}
	if config.Parameters["batch_size"] != 100 {
		t.Errorf("Parameters[batch_size] = %v, want %v", config.Parameters["batch_size"], 100)
	}
}

func TestEmbeddingProviderConstants(t *testing.T) {
	tests := []struct {
		name     string
		provider EmbeddingProvider
		expected string
	}{
		{"OpenAI", EmbeddingProviderOpenAI, "openai"},
		{"Azure", EmbeddingProviderAzure, "azure"},
		{"Cohere", EmbeddingProviderCohere, "cohere"},
		{"HuggingFace", EmbeddingProviderHuggingFace, "huggingface"},
		{"Bedrock", EmbeddingProviderBedrock, "bedrock"},
		{"Qwen", EmbeddingProviderQwen, "qwen"},
		{"DeepSeek", EmbeddingProviderDeepSeek, "deepseek"},
		{"Ollama", EmbeddingProviderOllama, "ollama"},
		{"Local", EmbeddingProviderLocal, "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.provider, tt.expected)
			}
		})
	}
}

// MockEmbeddingFunction for testing purposes
type MockEmbeddingFunction struct {
	dimension int
	name      string
}

func NewMockEmbeddingFunction(dimension int) *MockEmbeddingFunction {
	return &MockEmbeddingFunction{
		dimension: dimension,
		name:      "mock",
	}
}

func (f *MockEmbeddingFunction) EmbedDocuments(ctx interface{}, documents []string) ([][]float32, error) {
	// Return mock embeddings
	embeddings := make([][]float32, len(documents))
	for i := range embeddings {
		embeddings[i] = make([]float32, f.dimension)
		for j := range embeddings[i] {
			embeddings[i][j] = float32(i + j)
		}
	}
	return embeddings, nil
}

func (f *MockEmbeddingFunction) EmbedQuery(ctx interface{}, query string) ([]float32, error) {
	emb := make([]float32, f.dimension)
	for i := range emb {
		emb[i] = float32(i)
	}
	return emb, nil
}

func (f *MockEmbeddingFunction) Name() string {
	return f.name
}

func (f *MockEmbeddingFunction) Dimension() int {
	return f.dimension
}

func TestMockEmbeddingFunction(t *testing.T) {
	fn := NewMockEmbeddingFunction(128)

	if fn.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "mock")
	}
	if fn.Dimension() != 128 {
		t.Errorf("Dimension() = %d, want %d", fn.Dimension(), 128)
	}
}