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

// DashScope (Qwen) Embedding Function
// See: https://help.aliyun.com/zh/dashscope/developer-reference/text-embedding

// DashScopeEmbeddingRequest represents the request for DashScope embeddings.
type DashScopeEmbeddingRequest struct {
	Model string `json:"model"`
	Input struct {
		Texts []string `json:"texts"`
	} `json:"input"`
	Parameters struct {
		TextType string `json:"text_type,omitempty"`
	} `json:"parameters,omitempty"`
}

// DashScopeEmbeddingResponse represents the response from DashScope.
type DashScopeEmbeddingResponse struct {
	Output struct {
		Embeddings []struct {
			TextIndex int       `json:"text_index"`
			Embedding []float32 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// DashScopeEmbeddingFunction implements EmbeddingFunction for Alibaba Cloud DashScope/Qwen.
type DashScopeEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
	model     string
}

// NewDashScopeEmbeddingFunction creates a new DashScope embedding function.
func NewDashScopeEmbeddingFunction(model, apiKey string) *DashScopeEmbeddingFunction {
	// Default dimensions for Qwen models
	dimension := 1024
	if model == "text-embedding-v2" {
		dimension = 1536
	} else if model == "text-embedding-v3" {
		dimension = 1024
	}

	return &DashScopeEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "qwen",
			Model:    model,
			APIKey:   apiKey,
		},
		dimension: dimension,
		model:     model,
	}
}

func (f *DashScopeEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	reqBody := DashScopeEmbeddingRequest{
		Model: f.model,
	}
	reqBody.Input.Texts = documents
	reqBody.Parameters.TextType = "document"

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding", bytes.NewReader(jsonBody))
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

	var embeddingResp DashScopeEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embeddingResp.Code != "" {
		return nil, fmt.Errorf("DashScope API error: %s (code: %s)", embeddingResp.Message, embeddingResp.Code)
	}

	// Sort embeddings by text_index
	embeddings := make([][]float32, len(documents))
	for _, emb := range embeddingResp.Output.Embeddings {
		if emb.TextIndex < len(embeddings) {
			embeddings[emb.TextIndex] = emb.Embedding
		}
	}

	return embeddings, nil
}

func (f *DashScopeEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

func (f *DashScopeEmbeddingFunction) Name() string {
	return "qwen/" + f.model
}

func (f *DashScopeEmbeddingFunction) Dimension() int {
	return f.dimension
}

// DeepSeek Embedding Function

// DeepSeekEmbeddingFunction implements EmbeddingFunction for DeepSeek.
type DeepSeekEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
}

// NewDeepSeekEmbeddingFunction creates a new DeepSeek embedding function.
func NewDeepSeekEmbeddingFunction(apiKey string) *DeepSeekEmbeddingFunction {
	return &DeepSeekEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "deepseek",
			Model:    "deepseek-embedding",
			APIKey:   apiKey,
		},
		dimension: 1536,
	}
}

func (f *DeepSeekEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.deepseek.com/v1/embeddings", bytes.NewReader(jsonBody))
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
		return nil, fmt.Errorf("DeepSeek API error: %s", embeddingResp.Error.Message)
	}

	embeddings := make([][]float32, len(documents))
	for _, data := range embeddingResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

func (f *DeepSeekEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

func (f *DeepSeekEmbeddingFunction) Name() string {
	return "deepseek/" + f.config.Model
}

func (f *DeepSeekEmbeddingFunction) Dimension() int {
	return f.dimension
}

// Jina Embedding Function

// JinaEmbeddingRequest represents the request for Jina embeddings.
type JinaEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// JinaEmbeddingResponse represents the response from Jina.
type JinaEmbeddingResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Detail string `json:"detail,omitempty"`
}

// JinaEmbeddingFunction implements EmbeddingFunction for Jina AI.
type JinaEmbeddingFunction struct {
	config    EmbeddingFunctionConfig
	dimension int
	model     string
}

// NewJinaEmbeddingFunction creates a new Jina embedding function.
func NewJinaEmbeddingFunction(model, apiKey string) *JinaEmbeddingFunction {
	// Jina models dimensions
	dimension := 768
	if model == "jina-embeddings-v2-base-en" {
		dimension = 768
	} else if model == "jina-embeddings-v2-large-en" {
		dimension = 1024
	} else if model == "jina-colbert-v2-en" {
		dimension = 128 // multivector
	}

	return &JinaEmbeddingFunction{
		config: EmbeddingFunctionConfig{
			Provider: "jina",
			Model:    model,
			APIKey:   apiKey,
		},
		dimension: dimension,
		model:     model,
	}
}

func (f *JinaEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	reqBody := JinaEmbeddingRequest{
		Model: f.model,
		Input: documents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.jina.ai/v1/embeddings", bytes.NewReader(jsonBody))
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

	var embeddingResp JinaEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embeddingResp.Detail != "" {
		return nil, fmt.Errorf("Jina API error: %s", embeddingResp.Detail)
	}

	embeddings := make([][]float32, len(documents))
	for _, data := range embeddingResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

func (f *JinaEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := f.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

func (f *JinaEmbeddingFunction) Name() string {
	return "jina/" + f.model
}

func (f *JinaEmbeddingFunction) Dimension() int {
	return f.dimension
}