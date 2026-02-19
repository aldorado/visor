package memory

import (
	"encoding/json"
	"testing"
)

func TestEmbeddingResponseParse(t *testing.T) {
	raw := `{
		"data": [
			{"index": 0, "embedding": [0.1, 0.2, 0.3]},
			{"index": 1, "embedding": [0.4, 0.5, 0.6]}
		]
	}`

	var resp embeddingResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].Index != 0 {
		t.Errorf("data[0].index = %d, want 0", resp.Data[0].Index)
	}
	if len(resp.Data[0].Embedding) != 3 {
		t.Errorf("embedding len = %d, want 3", len(resp.Data[0].Embedding))
	}
	if resp.Data[1].Embedding[0] != 0.4 {
		t.Errorf("data[1].embedding[0] = %f, want 0.4", resp.Data[1].Embedding[0])
	}
}

func TestEmbeddingResponseParse_OutOfOrder(t *testing.T) {
	// API can return embeddings in any order — index field resolves position
	raw := `{
		"data": [
			{"index": 1, "embedding": [0.4, 0.5]},
			{"index": 0, "embedding": [0.1, 0.2]}
		]
	}`

	var resp embeddingResponse
	json.Unmarshal([]byte(raw), &resp)

	vectors := make([][]float32, len(resp.Data))
	for _, d := range resp.Data {
		vectors[d.Index] = d.Embedding
	}

	if vectors[0][0] != 0.1 {
		t.Errorf("vectors[0][0] = %f, want 0.1", vectors[0][0])
	}
	if vectors[1][0] != 0.4 {
		t.Errorf("vectors[1][0] = %f, want 0.4", vectors[1][0])
	}
}
