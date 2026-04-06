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

// OpenAIEmbeddingRequest represents the request body for OpenAI embeddings API.
type OpenAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// OpenAIEmbeddingResponse represents the response from OpenAI embeddings API.
type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// EmbedDocuments generates embeddings for multiple documents using OpenAI API.
func (f *OpenAIEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	reqBody := OpenAIEmbeddingRequest{
		Model: f.config.Model,
		Input: documents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.config.APIKey))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embeddingResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embeddingResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			embeddingResp.Error.Message, embeddingResp.Error.Type, embeddingResp.Error.Code)
	}

	// Sort embeddings by index to ensure correct order
	embeddings := make([][]float32, len(documents))
	for _, data := range embeddingResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

// EmbedQuery generates an embedding for a single query.
func (f *OpenAIEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

// OpenAIModels returns the available OpenAI embedding models and their dimensions.
func OpenAIModels() map[string]int {
	return map[string]int{
		"text-embedding-3-small": 1536,
		"text-embedding-3-large": 3072,
		"text-embedding-ada-002": 1536,
	}
}

// AzureOpenAIEmbeddingFunction implements EmbeddingFunction for Azure OpenAI.
type AzureOpenAIEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
	endpoint  string
	apiKey    string
}

// NewAzureOpenAIEmbeddingFunction creates a new Azure OpenAI embedding function.
func NewAzureOpenAIEmbeddingFunction(endpoint, deployment, apiKey string, dimension int) *AzureOpenAIEmbeddingFunction {
	return &AzureOpenAIEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "azure",
			Model:    deployment,
			APIKey:   apiKey,
			Parameters: map[string]interface{}{
				"endpoint": endpoint,
			},
		},
		dimension: dimension,
		endpoint:  endpoint,
		apiKey:    apiKey,
	}
}

func (f *AzureOpenAIEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	reqBody := OpenAIEmbeddingRequest{
		Model: f.config.Model,
		Input: documents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Azure OpenAI uses a different URL format
	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=2024-02-01", f.endpoint, f.config.Model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", f.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embeddingResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embeddingResp.Error != nil {
		return nil, fmt.Errorf("Azure OpenAI API error: %s", embeddingResp.Error.Message)
	}

	embeddings := make([][]float32, len(documents))
	for _, data := range embeddingResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

func (f *AzureOpenAIEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

func (f *AzureOpenAIEmbeddingFunction) Name() string {
	return "azure/" + f.config.Model
}

func (f *AzureOpenAIEmbeddingFunction) Dimension() int {
	return f.dimension
}

// CohereEmbeddingFunction implements EmbeddingFunction for Cohere.
type CohereEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
	model     string
}

// NewCohereEmbeddingFunction creates a new Cohere embedding function.
func NewCohereEmbeddingFunction(model, apiKey string) *CohereEmbeddingFunction {
	// Cohere embed-english-v3.0 has 1024 dimensions
	dimension := 1024
	if model == "embed-multilingual-v3.0" {
		dimension = 1024
	}

	return &CohereEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "cohere",
			Model:    model,
			APIKey:   apiKey,
		},
		dimension: dimension,
		model:     model,
	}
}

// CohereEmbeddingRequest represents the request for Cohere embeddings.
type CohereEmbeddingRequest struct {
	Texts     []string `json:"texts"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"`
}

// CohereEmbeddingResponse represents the response from Cohere.
type CohereEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	ID         string      `json:"id"`
	Texts      []string    `json:"texts"`
	Message    string      `json:"message,omitempty"`
}

func (f *CohereEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	reqBody := CohereEmbeddingRequest{
		Texts:     documents,
		Model:     f.model,
		InputType: "search_document",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cohere.ai/v1/embed", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.config.APIKey))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embeddingResp CohereEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embeddingResp.Message != "" {
		return nil, fmt.Errorf("Cohere API error: %s", embeddingResp.Message)
	}

	return embeddingResp.Embeddings, nil
}

func (f *CohereEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

func (f *CohereEmbeddingFunction) Name() string {
	return "cohere/" + f.model
}

func (f *CohereEmbeddingFunction) Dimension() int {
	return f.dimension
}