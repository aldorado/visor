package memory

import (
	"math"
	"testing"
)

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1, 0, 0}
	sim := cosineSimilarity(a, a)
	if math.Abs(sim-1.0) > 1e-6 {
		t.Errorf("identical vectors should have similarity 1.0, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim) > 1e-6 {
		t.Errorf("orthogonal vectors should have similarity 0, got %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{-1, 0}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim-(-1.0)) > 1e-6 {
		t.Errorf("opposite vectors should have similarity -1, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("different length vectors should return 0, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 0, 0}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("zero vector should return 0, got %f", sim)
	}
}

func TestSearch_RankedBySimilarity(t *testing.T) {
	memories := []Memory{
		{Text: "about dogs", Embedding: []float32{1, 0, 0}},
		{Text: "about cats", Embedding: []float32{0.9, 0.1, 0}},
		{Text: "about fish", Embedding: []float32{0, 0, 1}},
	}
	query := []float32{1, 0, 0} // most similar to "about dogs"

	results := Search(memories, query, 3, 0, 0)
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Memory.Text != "about dogs" {
		t.Errorf("first result = %q, want 'about dogs'", results[0].Memory.Text)
	}
	if results[1].Memory.Text != "about cats" {
		t.Errorf("second result = %q, want 'about cats'", results[1].Memory.Text)
	}
}

func TestSearch_MaxResults(t *testing.T) {
	memories := []Memory{
		{Text: "a", Embedding: []float32{1, 0}},
		{Text: "b", Embedding: []float32{0.9, 0.1}},
		{Text: "c", Embedding: []float32{0.8, 0.2}},
	}
	results := Search(memories, []float32{1, 0}, 2, 0, 0)
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestSearch_MinResults(t *testing.T) {
	memories := []Memory{
		{Text: "a", Embedding: []float32{1, 0}},
		{Text: "b", Embedding: []float32{0, 1}},
	}
	// high threshold that nothing passes, but minResults=2
	results := Search(memories, []float32{0.5, 0.5}, 10, 2, 0.99)
	if len(results) < 2 {
		t.Errorf("got %d results, want at least 2 (minResults guarantee)", len(results))
	}
}

func TestSearch_SkipsEmptyEmbeddings(t *testing.T) {
	memories := []Memory{
		{Text: "has embedding", Embedding: []float32{1, 0}},
		{Text: "no embedding", Embedding: nil},
	}
	results := Search(memories, []float32{1, 0}, 10, 0, 0)
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (skip empty embeddings)", len(results))
	}
}

func TestSearch_ThresholdFilter(t *testing.T) {
	memories := []Memory{
		{Text: "close", Embedding: []float32{1, 0}},
		{Text: "far", Embedding: []float32{0, 1}},
	}
	results := Search(memories, []float32{1, 0}, 10, 0, 0.5)
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (threshold filter)", len(results))
	}
	if results[0].Memory.Text != "close" {
		t.Errorf("expected 'close', got %q", results[0].Memory.Text)
	}
}
