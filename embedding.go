package seekdb

import (
	"context"
)

// EmbeddingFunction defines the interface for embedding functions.
// Embedding functions convert text into vector embeddings.
type EmbeddingFunction interface {
	// EmbedDocuments generates embeddings for multiple documents.
	EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error)

	// EmbedQuery generates an embedding for a single query.
	EmbedQuery(ctx context.Context, query string) ([]float32, error)

	// Name returns the name of the embedding function.
	Name() string

	// Dimension returns the dimension of the embeddings.
	Dimension() int
}

// EmbeddingFunctionConfig holds the configuration for an embedding function.
type EmbeddingFunctionConfig struct {
	Provider   string                 // e.g., "openai", "cohere", "huggingface"
	Model      string                 // e.g., "text-embedding-3-small"
	APIKey     string                 // API key for the provider
	Parameters map[string]interface{} // Additional parameters
}

// OpenAIEmbeddingFunction implements EmbeddingFunction for OpenAI.
type OpenAIEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
}

// NewOpenAIEmbeddingFunction creates a new OpenAI embedding function.
func NewOpenAIEmbeddingFunction(model string, apiKey string) *OpenAIEmbeddingFunction {
	dimension := 1536 // default for text-embedding-3-small
	if model == "text-embedding-3-large" {
		dimension = 3072
	}

	return &OpenAIEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "openai",
			Model:    model,
			APIKey:   apiKey,
		},
		dimension: dimension,
	}
}

func (f *OpenAIEmbeddingFunction) Name() string {
	return "openai/" + f.config.Model
}

func (f *OpenAIEmbeddingFunction) Dimension() int {
	return f.dimension
}

// HuggingFaceEmbeddingFunction implements EmbeddingFunction for HuggingFace.
type HuggingFaceEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
}

// NewHuggingFaceEmbeddingFunction creates a new HuggingFace embedding function.
func NewHuggingFaceEmbeddingFunction(model string, apiKey string) *HuggingFaceEmbeddingFunction {
	return &HuggingFaceEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "huggingface",
			Model:    model,
			APIKey:   apiKey,
		},
		dimension: 768, // default dimension
	}
}

func (f *HuggingFaceEmbeddingFunction) Name() string {
	return "huggingface/" + f.config.Model
}

func (f *HuggingFaceEmbeddingFunction) Dimension() int {
	return f.dimension
}

// EmbeddingProvider represents supported embedding providers.
type EmbeddingProvider string

const (
	EmbeddingProviderOpenAI      EmbeddingProvider = "openai"
	EmbeddingProviderAzure       EmbeddingProvider = "azure"
	EmbeddingProviderCohere      EmbeddingProvider = "cohere"
	EmbeddingProviderHuggingFace EmbeddingProvider = "huggingface"
	EmbeddingProviderBedrock     EmbeddingProvider = "bedrock"
	EmbeddingProviderQwen        EmbeddingProvider = "qwen"
	EmbeddingProviderDeepSeek    EmbeddingProvider = "deepseek"
	EmbeddingProviderOllama      EmbeddingProvider = "ollama"
	EmbeddingProviderLocal       EmbeddingProvider = "local"
)
