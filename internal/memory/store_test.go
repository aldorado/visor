package memory

import (
	"os"
	"testing"
	"time"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "visor-memory-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestStore_AppendAndReadAll(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Append([]Memory{
		{Text: "first memory"},
		{Text: "second memory"},
	})
	if err != nil {
		t.Fatal(err)
	}

	all, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d memories, want 2", len(all))
	}
	if all[0].Text != "first memory" {
		t.Errorf("all[0].Text = %q, want %q", all[0].Text, "first memory")
	}
	if all[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if all[0].CreatedAt == 0 {
		t.Error("expected auto-generated timestamp")
	}
}

func TestStore_MultipleChunks(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	store.Append([]Memory{{Text: "chunk1", CreatedAt: 100}})
	time.Sleep(time.Millisecond) // ensure different chunk filenames
	store.Append([]Memory{{Text: "chunk2", CreatedAt: 200}})

	all, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d, want 2", len(all))
	}
	// should be sorted by created_at
	if all[0].Text != "chunk1" || all[1].Text != "chunk2" {
		t.Errorf("unexpected order: %v", all)
	}
}

func TestStore_FilterByDate(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	store.Append([]Memory{
		{Text: "old", CreatedAt: 1000},
		{Text: "mid", CreatedAt: 2000},
		{Text: "new", CreatedAt: 3000},
	})

	filtered, err := store.FilterByDate(1500, 2500)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Text != "mid" {
		t.Errorf("got %v, want [mid]", filtered)
	}
}

func TestStore_Count(t *testing.T) {
	dir := tempDir(t)
	store, _ := NewStore(dir)

	count, _ := store.Count()
	if count != 0 {
		t.Errorf("empty store count = %d, want 0", count)
	}

	store.Append([]Memory{{Text: "one"}, {Text: "two"}, {Text: "three"}})
	count, _ = store.Count()
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestStore_Compact(t *testing.T) {
	dir := tempDir(t)
	store, _ := NewStore(dir)

	store.Append([]Memory{{Text: "a", CreatedAt: 100}})
	time.Sleep(time.Millisecond)
	store.Append([]Memory{{Text: "b", CreatedAt: 200}})
	time.Sleep(time.Millisecond)
	store.Append([]Memory{{Text: "c", CreatedAt: 300}})

	// should have 3 chunk files
	chunks, _ := store.listChunks()
	if len(chunks) != 3 {
		t.Fatalf("pre-compact chunks = %d, want 3", len(chunks))
	}

	if err := store.Compact(); err != nil {
		t.Fatal(err)
	}

	// should have 1 chunk file after compaction
	chunks, _ = store.listChunks()
	if len(chunks) != 1 {
		t.Fatalf("post-compact chunks = %d, want 1", len(chunks))
	}

	// all memories should still be there
	all, _ := store.ReadAll()
	if len(all) != 3 {
		t.Fatalf("post-compact memories = %d, want 3", len(all))
	}
	if all[0].Text != "a" || all[2].Text != "c" {
		t.Error("post-compact order wrong")
	}
}

func TestStore_AppendEmpty(t *testing.T) {
	dir := tempDir(t)
	store, _ := NewStore(dir)
	if err := store.Append(nil); err != nil {
		t.Errorf("appending nil should not error: %v", err)
	}
}

func TestStore_WithEmbedding(t *testing.T) {
	dir := tempDir(t)
	store, _ := NewStore(dir)

	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	store.Append([]Memory{{Text: "with embedding", Embedding: embedding}})

	all, _ := store.ReadAll()
	if len(all) != 1 {
		t.Fatal("expected 1 memory")
	}
	if len(all[0].Embedding) != 1536 {
		t.Errorf("embedding len = %d, want 1536", len(all[0].Embedding))
	}
	if all[0].Embedding[100] != 0.1 {
		t.Errorf("embedding[100] = %f, want 0.1", all[0].Embedding[100])
	}
}
