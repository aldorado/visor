package memory

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
)

// Memory is a single memory entry stored in parquet.
type Memory struct {
	ID        string    `parquet:"id"`
	Text      string    `parquet:"text"`
	Embedding []float32 `parquet:"embedding,list"`
	CreatedAt int64     `parquet:"created_at"` // unix millis
}

// Store manages persistent memories in parquet files.
// Uses append-by-new-file strategy: each write creates a new chunk file.
// Periodic compaction merges chunks into a single file.
type Store struct {
	dir string
	mu  sync.RWMutex
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("memory: create dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Append writes new memories as a new parquet chunk file.
func (s *Store) Append(memories []Memory) error {
	if len(memories) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range memories {
		if memories[i].ID == "" {
			memories[i].ID = uuid.New().String()
		}
		if memories[i].CreatedAt == 0 {
			memories[i].CreatedAt = time.Now().UnixMilli()
		}
	}

	filename := fmt.Sprintf("chunk_%d.parquet", time.Now().UnixNano())
	path := filepath.Join(s.dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("memory: create chunk: %w", err)
	}

	w := parquet.NewGenericWriter[Memory](f)
	if _, err := w.Write(memories); err != nil {
		f.Close()
		return fmt.Errorf("memory: write rows: %w", err)
	}
	if err := w.Close(); err != nil {
		f.Close()
		return fmt.Errorf("memory: close writer: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("memory: sync file: %w", err)
	}
	return f.Close()
}

// ReadAll loads all memories from all chunk files, sorted by created_at ascending.
func (s *Store) ReadAll() ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chunks, err := s.listChunks()
	if err != nil {
		return nil, err
	}

	var all []Memory
	for _, path := range chunks {
		memories, err := s.readChunk(path)
		if err != nil {
			return nil, fmt.Errorf("memory: read %s: %w", path, err)
		}
		all = append(all, memories...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt < all[j].CreatedAt
	})
	return all, nil
}

// FilterByDate returns memories created between start and end (inclusive, unix millis).
func (s *Store) FilterByDate(startMillis, endMillis int64) ([]Memory, error) {
	all, err := s.ReadAll()
	if err != nil {
		return nil, err
	}
	var filtered []Memory
	for _, m := range all {
		if m.CreatedAt >= startMillis && m.CreatedAt <= endMillis {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// Count returns the total number of memories.
func (s *Store) Count() (int, error) {
	all, err := s.ReadAll()
	if err != nil {
		return 0, err
	}
	return len(all), nil
}

// Compact merges all chunk files into a single parquet file.
func (s *Store) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chunks, err := s.listChunks()
	if err != nil {
		return err
	}
	if len(chunks) <= 1 {
		return nil
	}

	var all []Memory
	for _, path := range chunks {
		memories, err := s.readChunk(path)
		if err != nil {
			return fmt.Errorf("memory: compact read %s: %w", path, err)
		}
		all = append(all, memories...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt < all[j].CreatedAt
	})

	compacted := filepath.Join(s.dir, "compacted.parquet.tmp")
	f, err := os.Create(compacted)
	if err != nil {
		return fmt.Errorf("memory: create compacted: %w", err)
	}

	w := parquet.NewGenericWriter[Memory](f)
	if _, err := w.Write(all); err != nil {
		f.Close()
		os.Remove(compacted)
		return fmt.Errorf("memory: write compacted: %w", err)
	}
	if err := w.Close(); err != nil {
		f.Close()
		os.Remove(compacted)
		return fmt.Errorf("memory: close compacted: %w", err)
	}
	f.Close()

	// remove old chunks
	for _, path := range chunks {
		os.Remove(path)
	}

	// rename compacted file
	final := filepath.Join(s.dir, fmt.Sprintf("chunk_%d.parquet", time.Now().UnixNano()))
	if err := os.Rename(compacted, final); err != nil {
		return fmt.Errorf("memory: rename compacted: %w", err)
	}
	return nil
}

func (s *Store) listChunks() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("memory: list dir: %w", err)
	}
	var chunks []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".parquet" {
			chunks = append(chunks, filepath.Join(s.dir, e.Name()))
		}
	}
	sort.Strings(chunks)
	return chunks, nil
}

func (s *Store) readChunk(path string) ([]Memory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return nil, err
	}

	r := parquet.NewGenericReader[Memory](pf)
	defer r.Close()

	memories := make([]Memory, r.NumRows())
	n, err := r.Read(memories)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return memories[:n], nil
}
