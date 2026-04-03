package internal

import "context"

// VectorStoreProvider abstracts vector database operations across backends.
type VectorStoreProvider interface {
	Connect(ctx context.Context, config ProviderConfig) error
	Upsert(ctx context.Context, vectors []Vector) error
	Query(ctx context.Context, vector []float32, topK int, filter map[string]any) ([]Match, error)
	Fetch(ctx context.Context, ids []string) ([]Vector, error)
	Delete(ctx context.Context, ids []string) error
	DeleteByFilter(ctx context.Context, filter map[string]any) error
	CreateIndex(ctx context.Context, cfg IndexConfig) error
	ListIndexes(ctx context.Context) ([]IndexInfo, error)
	DescribeIndex(ctx context.Context, name string) (*IndexInfo, error)
	Close() error
}

// Vector represents a single vector with metadata.
type Vector struct {
	ID       string
	Values   []float32
	Metadata map[string]any
}

// Match represents a search result with similarity score.
type Match struct {
	ID       string
	Score    float32
	Metadata map[string]any
	Values   []float32
}

// ProviderConfig holds connection settings for a vector store backend.
type ProviderConfig struct {
	Provider    string
	APIKey      string
	Environment string
	Host        string
	Namespace   string
	IndexName   string
}

// IndexConfig holds parameters for creating a new vector index.
type IndexConfig struct {
	Name      string
	Dimension int
	Metric    string
	Cloud     string
	Region    string
}

// IndexInfo describes an existing vector index.
type IndexInfo struct {
	Name        string
	Dimension   int
	Metric      string
	VectorCount int64
}
