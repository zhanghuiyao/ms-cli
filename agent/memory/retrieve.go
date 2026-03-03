package memory

// RetrieveResult is a memory search result with relevance score.
type RetrieveResult struct {
	Key   string
	Value string
	Score int
}
