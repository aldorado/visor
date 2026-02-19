package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	openAIEmbeddingsURL = "https://api.openai.com/v1/embeddings"
	embeddingModel      = "text-embedding-3-small"
	embeddingDims       = 1536
)

type EmbeddingClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewEmbeddingClient(apiKey string) *EmbeddingClient {
	return &EmbeddingClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Embed generates an embedding vector for a single text.
func (c *EmbeddingClient) Embed(text string) ([]float32, error) {
	vectors, err := c.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts in a single API call.
func (c *EmbeddingClient) EmbedBatch(texts []string) ([][]float32, error) {
	payload := map[string]any{
		"model": embeddingModel,
		"input": texts,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("embeddings: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", openAIEmbeddingsURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embeddings: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings: status %d: %s", resp.StatusCode, respBody)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("embeddings: decode: %w", err)
	}

	vectors := make([][]float32, len(result.Data))
	for _, d := range result.Data {
		vectors[d.Index] = d.Embedding
	}
	return vectors, nil
}

type embeddingResponse struct {
	Data []embeddingData `json:"data"`
}

type embeddingData struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}
