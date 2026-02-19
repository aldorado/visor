package memory

import (
	"math"
	"sort"
)

type SearchResult struct {
	Memory     Memory
	Similarity float64
}

// Search finds the most similar memories to the query embedding.
// Returns up to maxResults, filtered by minSimilarity threshold.
// If fewer than minResults meet the threshold, returns top minResults anyway.
func Search(memories []Memory, queryEmbedding []float32, maxResults, minResults int, minSimilarity float64) []SearchResult {
	var results []SearchResult

	for _, m := range memories {
		if len(m.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(queryEmbedding, m.Embedding)
		results = append(results, SearchResult{Memory: m, Similarity: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// filter by threshold
	var filtered []SearchResult
	for _, r := range results {
		if r.Similarity >= minSimilarity {
			filtered = append(filtered, r)
		}
		if len(filtered) >= maxResults {
			break
		}
	}

	// guarantee minResults even below threshold
	if len(filtered) < minResults && len(results) > len(filtered) {
		for _, r := range results[len(filtered):] {
			filtered = append(filtered, r)
			if len(filtered) >= minResults {
				break
			}
		}
	}

	return filtered
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
