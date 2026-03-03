package memory

import (
	"testing"
)

func TestSimpleEmbedder(t *testing.T) {
	embedder := NewSimpleEmbedder()

	// Test embedding
	embedding, err := embedder.Embed("hello world")
	if err != nil {
		t.Fatal(err)
	}

	if len(embedding) == 0 {
		t.Error("embedding should not be empty")
	}

	// Should be normalized (length ≈ 1)
	var sum float64
	for _, v := range embedding {
		sum += float64(v * v)
	}
	if sum < 0.9 || sum > 1.1 {
		t.Errorf("embedding should be normalized, got length %f", sum)
	}
}

func TestVectorStore(t *testing.T) {
	embedder := NewSimpleEmbedder()
	store := NewVectorStore(embedder)

	// Add items
	err := store.Add("doc1", "The quick brown fox", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Add("doc2", "The lazy dog sleeps", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Add("doc3", "Programming in Go", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Check count
	if store.Count() != 3 {
		t.Errorf("expected 3 items, got %d", store.Count())
	}

	// Search
	results, err := store.Search("fox", 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Error("should find results")
	}

	// Best match should be doc1 (contains "fox")
	if results[0].Key != "doc1" {
		t.Errorf("expected doc1 as best match, got %s", results[0].Key)
	}

	if results[0].Score <= 0 {
		t.Error("score should be positive")
	}
}

func TestVectorStoreDelete(t *testing.T) {
	embedder := NewSimpleEmbedder()
	store := NewVectorStore(embedder)

	store.Add("doc1", "Content 1", nil)
	store.Add("doc2", "Content 2", nil)

	if store.Count() != 2 {
		t.Error("expected 2 items")
	}

	// Delete
	if !store.Delete("doc1") {
		t.Error("delete should return true")
	}

	if store.Count() != 1 {
		t.Errorf("expected 1 item, got %d", store.Count())
	}

	// Delete non-existent
	if store.Delete("doc3") {
		t.Error("delete should return false for non-existent")
	}
}

func TestVectorStoreClear(t *testing.T) {
	embedder := NewSimpleEmbedder()
	store := NewVectorStore(embedder)

	store.Add("doc1", "Content 1", nil)
	store.Add("doc2", "Content 2", nil)

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("expected 0 items after clear, got %d", store.Count())
	}
}

func TestCosineSimilarity(t *testing.T) {
	// Test identical vectors
	v1 := []float32{1, 0, 0}
	v2 := []float32{1, 0, 0}
	sim := cosineSimilarity(v1, v2)
	if sim < 0.99 {
		t.Errorf("identical vectors should have similarity ~1, got %f", sim)
	}

	// Test orthogonal vectors
	v3 := []float32{0, 1, 0}
	sim = cosineSimilarity(v1, v3)
	if sim > 0.01 {
		t.Errorf("orthogonal vectors should have similarity ~0, got %f", sim)
	}

	// Test different length vectors
	v4 := []float32{1, 0}
	sim = cosineSimilarity(v1, v4)
	if sim != 0 {
		t.Error("different length vectors should return 0")
	}
}
