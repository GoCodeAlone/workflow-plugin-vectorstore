package internal

import (
	"context"
	"fmt"
)

// MilvusAdapter is a placeholder for Milvus vector database support.
// TODO: Implement with github.com/milvus-io/milvus/client/v2 once transitive
// dependency issues (etcd/otelgrpc incompatibility) are resolved upstream.
type MilvusAdapter struct{}

var _ VectorStoreProvider = (*MilvusAdapter)(nil)

func (a *MilvusAdapter) Connect(_ context.Context, _ ProviderConfig) error {
	return fmt.Errorf("milvus adapter is not yet implemented — use pinecone")
}

func (a *MilvusAdapter) Upsert(_ context.Context, _ []Vector) error {
	return fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) Query(_ context.Context, _ []float32, _ int, _ map[string]any) ([]Match, error) {
	return nil, fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) Fetch(_ context.Context, _ []string) ([]Vector, error) {
	return nil, fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) Delete(_ context.Context, _ []string) error {
	return fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) DeleteByFilter(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) CreateIndex(_ context.Context, _ IndexConfig) error {
	return fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) ListIndexes(_ context.Context) ([]IndexInfo, error) {
	return nil, fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) DescribeIndex(_ context.Context, _ string) (*IndexInfo, error) {
	return nil, fmt.Errorf("milvus: not implemented")
}

func (a *MilvusAdapter) Close() error { return nil }
