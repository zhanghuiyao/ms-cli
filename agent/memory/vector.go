package memory

import (
	"math"
	"sort"
	"strings"
)

// VectorStore provides semantic search using vector embeddings.
type VectorStore struct {
	items    []VectorItem
	embedder Embedder
}

// VectorItem represents an item with its vector embedding.
type VectorItem struct {
	Key       string
	Value     string
	Embedding []float32
	Metadata  map[string]string
}

// Embedder generates vector embeddings for text.
type Embedder interface {
	Embed(text string) ([]float32, error)
}

// SimpleEmbedder is a basic embedder using bag-of-words.
// In production, this would use a real embedding model.
type SimpleEmbedder struct {
	vocab map[string]int
	dim   int
}

// NewSimpleEmbedder creates a simple embedder.
func NewSimpleEmbedder() *SimpleEmbedder {
	return &SimpleEmbedder{
		vocab: make(map[string]int),
		dim:   100,
	}
}

// Embed generates a simple embedding for text.
func (e *SimpleEmbedder) Embed(text string) ([]float32, error) {
	// Tokenize
	words := tokenize(text)

	// Build vocabulary
	for _, word := range words {
		if _, ok := e.vocab[word]; !ok {
			e.vocab[word] = len(e.vocab)
		}
	}

	// Create bag-of-words vector
	embedding := make([]float32, e.dim)
	for _, word := range words {
		if idx, ok := e.vocab[word]; ok && idx < e.dim {
			embedding[idx] += 1.0
		}
	}

	// Normalize
	normalize(embedding)

	return embedding, nil
}

// NewVectorStore creates a new vector store.
func NewVectorStore(embedder Embedder) *VectorStore {
	return &VectorStore{
		items:    make([]VectorItem, 0),
		embedder: embedder,
	}
}

// Add adds an item to the store.
func (v *VectorStore) Add(key, value string, metadata map[string]string) error {
	embedding, err := v.embedder.Embed(value)
	if err != nil {
		return err
	}

	v.items = append(v.items, VectorItem{
		Key:       key,
		Value:     value,
		Embedding: embedding,
		Metadata:  metadata,
	})

	return nil
}

// Search finds items similar to the query.
func (v *VectorStore) Search(query string, topK int) ([]SearchResult, error) {
	if len(v.items) == 0 {
		return []SearchResult{}, nil
	}

	// Embed query
	queryEmbedding, err := v.embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	// Calculate similarities
	results := make([]SearchResult, 0, len(v.items))
	for _, item := range v.items {
		similarity := cosineSimilarity(queryEmbedding, item.Embedding)
		results = append(results, SearchResult{
			Key:        item.Key,
			Value:      item.Value,
			Score:      similarity,
			Metadata:   item.Metadata,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K
	if topK > 0 && topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

// Delete removes an item by key.
func (v *VectorStore) Delete(key string) bool {
	for i, item := range v.items {
		if item.Key == key {
			v.items = append(v.items[:i], v.items[i+1:]...)
			return true
		}
	}
	return false
}

// Count returns the number of items.
func (v *VectorStore) Count() int {
	return len(v.items)
}

// Clear removes all items.
func (v *VectorStore) Clear() {
	v.items = make([]VectorItem, 0)
}

// SearchResult represents a search result with relevance score.
type SearchResult struct {
	Key      string
	Value    string
	Score    float32
	Metadata map[string]string
}

// tokenize splits text into words.
func tokenize(text string) []string {
	// Simple tokenization
	words := strings.Fields(strings.ToLower(text))
	for i, word := range words {
		// Remove punctuation
		words[i] = strings.Trim(word, ".,!?;:\"'()[]{}\n\t")
	}
	return words
}

// normalize normalizes a vector to unit length.
func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range v {
		v[i] /= norm
	}
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}

	return float32(dot) // Already normalized
}
