package memory

import (
	"fmt"
	"log"
	"strings"
)

// Manager ties together storage, embeddings, and search.
type Manager struct {
	store    *Store
	embedder *EmbeddingClient
}

func NewManager(dataDir string, openAIKey string) (*Manager, error) {
	store, err := NewStore(dataDir + "/memories")
	if err != nil {
		return nil, err
	}
	return &Manager{
		store:    store,
		embedder: NewEmbeddingClient(openAIKey),
	}, nil
}

// Save embeds and stores new memories.
func (m *Manager) Save(texts []string) error {
	if len(texts) == 0 {
		return nil
	}

	embeddings, err := m.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("memory save: embed: %w", err)
	}

	memories := make([]Memory, len(texts))
	for i, text := range texts {
		memories[i] = Memory{
			Text:      text,
			Embedding: embeddings[i],
		}
	}

	if err := m.store.Append(memories); err != nil {
		return fmt.Errorf("memory save: store: %w", err)
	}

	log.Printf("memory: saved %d memories", len(texts))
	return nil
}

// Lookup searches memories relevant to a query and returns formatted context.
func (m *Manager) Lookup(query string, maxResults int) (string, error) {
	queryEmb, err := m.embedder.Embed(query)
	if err != nil {
		return "", fmt.Errorf("memory lookup: embed query: %w", err)
	}

	all, err := m.store.ReadAll()
	if err != nil {
		return "", fmt.Errorf("memory lookup: read: %w", err)
	}

	if len(all) == 0 {
		return "", nil
	}

	results := Search(all, queryEmb, maxResults, 3, 0.3)
	if len(results) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("relevant memories:\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- [%.2f] %s\n", r.Similarity, r.Memory.Text))
	}
	return sb.String(), nil
}

// Store returns the underlying memory store (for direct access if needed).
func (m *Manager) Store() *Store {
	return m.store
}
