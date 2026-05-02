package internal_test

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-vectorstore/internal"
)

// mockProvider is a fake VectorStoreProvider for unit tests.
type mockProvider struct {
	upserted []internal.Vector
	queried  []float32
	fetched  []string
	deleted  []string
	indexes  []internal.IndexInfo
}

func (m *mockProvider) Connect(_ context.Context, _ internal.ProviderConfig) error { return nil }

func (m *mockProvider) Upsert(_ context.Context, vectors []internal.Vector) error {
	m.upserted = append(m.upserted, vectors...)
	return nil
}

func (m *mockProvider) Query(_ context.Context, vector []float32, topK int, _ map[string]any) ([]internal.Match, error) {
	m.queried = vector
	matches := make([]internal.Match, 0, topK)
	for i := 0; i < topK && i < 3; i++ {
		matches = append(matches, internal.Match{
			ID:    "result-" + string(rune('a'+i)),
			Score: float32(0.9) - float32(i)*0.1,
			Metadata: map[string]any{
				"text": "sample",
			},
		})
	}
	return matches, nil
}

func (m *mockProvider) Fetch(_ context.Context, ids []string) ([]internal.Vector, error) {
	m.fetched = ids
	vectors := make([]internal.Vector, len(ids))
	for i, id := range ids {
		vectors[i] = internal.Vector{
			ID:     id,
			Values: []float32{0.1, 0.2, 0.3},
		}
	}
	return vectors, nil
}

func (m *mockProvider) Delete(_ context.Context, ids []string) error {
	m.deleted = ids
	return nil
}

func (m *mockProvider) DeleteByFilter(_ context.Context, _ map[string]any) error { return nil }

func (m *mockProvider) CreateIndex(_ context.Context, cfg internal.IndexConfig) error {
	m.indexes = append(m.indexes, internal.IndexInfo{
		Name:      cfg.Name,
		Dimension: cfg.Dimension,
		Metric:    cfg.Metric,
	})
	return nil
}

func (m *mockProvider) ListIndexes(_ context.Context) ([]internal.IndexInfo, error) {
	return m.indexes, nil
}

func (m *mockProvider) DescribeIndex(_ context.Context, name string) (*internal.IndexInfo, error) {
	for _, idx := range m.indexes {
		if idx.Name == name {
			return &idx, nil
		}
	}
	return &internal.IndexInfo{Name: name, Dimension: 1536, Metric: "cosine"}, nil
}

func (m *mockProvider) Close() error { return nil }

func setupMock(t *testing.T) *mockProvider {
	t.Helper()
	mp := &mockProvider{}
	internal.RegisterProvider("test-vs", mp)
	t.Cleanup(func() { internal.UnregisterProvider("test-vs") })
	return mp
}

func TestVectorUpsertStep(t *testing.T) {
	mp := setupMock(t)
	step := &internal.VectorUpsertStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"vectors": []any{
			map[string]any{
				"id":     "v1",
				"values": []any{0.1, 0.2, 0.3},
			},
			map[string]any{
				"id":     "v2",
				"values": []any{0.4, 0.5, 0.6},
			},
		},
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["upserted_count"] != 2 {
		t.Errorf("expected 2 upserted, got %v", result.Output["upserted_count"])
	}
	if len(mp.upserted) != 2 {
		t.Errorf("expected 2 vectors upserted, got %d", len(mp.upserted))
	}
}

func TestVectorQueryStep(t *testing.T) {
	setupMock(t)
	step := &internal.VectorQueryStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"vector": []any{0.1, 0.2, 0.3},
		"top_k":  5,
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matches, ok := result.Output["matches"].([]any)
	if !ok {
		t.Fatal("matches not []any")
	}
	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d", len(matches))
	}
	if result.Output["count"] != 3 {
		t.Errorf("expected count 3, got %v", result.Output["count"])
	}
}

func TestVectorFetchStep(t *testing.T) {
	mp := setupMock(t)
	step := &internal.VectorFetchStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"ids":    []any{"v1", "v2"},
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["count"] != 2 {
		t.Errorf("expected 2 vectors, got %v", result.Output["count"])
	}
	if len(mp.fetched) != 2 {
		t.Errorf("expected 2 fetched IDs, got %d", len(mp.fetched))
	}
}

func TestVectorDeleteStep_ByIDs(t *testing.T) {
	mp := setupMock(t)
	step := &internal.VectorDeleteStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"ids":    []any{"v1", "v2"},
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["deleted_count"] != 2 {
		t.Errorf("expected 2 deleted, got %v", result.Output["deleted_count"])
	}
	if len(mp.deleted) != 2 {
		t.Errorf("expected 2 deleted IDs, got %d", len(mp.deleted))
	}
}

func TestVectorDeleteStep_ByFilter(t *testing.T) {
	setupMock(t)
	step := &internal.VectorDeleteStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"filter": map[string]any{"category": "old"},
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["deleted_by_filter"] != true {
		t.Error("expected deleted_by_filter=true")
	}
}

func TestVectorDeleteStep_NoIDsOrFilter(t *testing.T) {
	setupMock(t)
	step := &internal.VectorDeleteStep{}
	cfg := map[string]any{"module": "test-vs"}

	_, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err == nil {
		t.Error("expected error when no ids or filter")
	}
}

func TestVectorDeleteStep_BothIDsAndFilter(t *testing.T) {
	setupMock(t)
	step := &internal.VectorDeleteStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"ids":    []any{"v1"},
		"filter": map[string]any{"category": "old"},
	}

	_, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err == nil {
		t.Error("expected error when both ids and filter are provided")
	}
}

func TestVectorCreateIndexStep(t *testing.T) {
	setupMock(t)
	step := &internal.VectorCreateIndexStep{}
	cfg := map[string]any{
		"module":    "test-vs",
		"name":      "test-index",
		"dimension": 1536,
		"metric":    "cosine",
		"cloud":     "aws",
		"region":    "us-east-1",
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["created"] != "test-index" {
		t.Errorf("expected created=test-index, got %v", result.Output["created"])
	}
}

func TestVectorListIndexesStep(t *testing.T) {
	mp := setupMock(t)
	mp.indexes = []internal.IndexInfo{
		{Name: "idx-1", Dimension: 768, Metric: "cosine"},
		{Name: "idx-2", Dimension: 1536, Metric: "dotproduct"},
	}

	step := &internal.VectorListIndexesStep{}
	cfg := map[string]any{"module": "test-vs"}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["count"] != 2 {
		t.Errorf("expected 2 indexes, got %v", result.Output["count"])
	}
}

func TestVectorDescribeIndexStep(t *testing.T) {
	setupMock(t)
	step := &internal.VectorDescribeIndexStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"name":   "my-index",
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["name"] != "my-index" {
		t.Errorf("expected name=my-index, got %v", result.Output["name"])
	}
	if result.Output["dimension"] != 1536 {
		t.Errorf("expected dimension=1536, got %v", result.Output["dimension"])
	}
}

func TestStep_MissingModule(t *testing.T) {
	step := &internal.VectorQueryStep{}
	cfg := map[string]any{
		"vector": []any{0.1, 0.2},
		"top_k":  5,
	}

	_, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err == nil {
		t.Error("expected error when module is missing")
	}
}

func TestStep_ModuleNotFound(t *testing.T) {
	step := &internal.VectorQueryStep{}
	cfg := map[string]any{
		"module": "nonexistent",
		"vector": []any{0.1, 0.2},
		"top_k":  5,
	}

	_, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err == nil {
		t.Error("expected error when module not found")
	}
}

func TestVectorUpsertStep_EmptyVectors(t *testing.T) {
	setupMock(t)
	step := &internal.VectorUpsertStep{}

	// Missing vectors key entirely
	cfg := map[string]any{"module": "test-vs"}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Output["error"] == nil {
		t.Error("expected error in output when vectors missing")
	}

	// Empty vectors slice
	cfg2 := map[string]any{"module": "test-vs", "vectors": []any{}}
	result2, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg2)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result2.Output["error"] == nil {
		t.Error("expected error in output when vectors empty")
	}
}

func TestVectorQueryStep_NegativeTopK(t *testing.T) {
	setupMock(t)
	step := &internal.VectorQueryStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"vector": []any{0.1, 0.2, 0.3},
		"top_k":  -5,
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Output["error"] == nil {
		t.Error("expected error in output for negative top_k")
	}
}

func TestVectorQueryStep_ZeroTopK(t *testing.T) {
	setupMock(t)
	step := &internal.VectorQueryStep{}
	cfg := map[string]any{
		"module": "test-vs",
		"vector": []any{0.1, 0.2, 0.3},
		"top_k":  0,
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Output["error"] == nil {
		t.Error("expected error in output for zero top_k")
	}
}

func TestVectorCreateIndexStep_InvalidCloud(t *testing.T) {
	setupMock(t)
	step := &internal.VectorCreateIndexStep{}
	cfg := map[string]any{
		"module":    "test-vs",
		"name":      "test-index",
		"dimension": 1536,
		"cloud":     "typo-cloud",
		"region":    "us-east-1",
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Output["error"] == nil {
		t.Error("expected error in output for invalid cloud")
	}
}

func TestVectorCreateIndexStep_EmptyCloud(t *testing.T) {
	setupMock(t)
	step := &internal.VectorCreateIndexStep{}
	cfg := map[string]any{
		"module":    "test-vs",
		"name":      "test-index",
		"dimension": 1536,
		"region":    "us-east-1",
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Output["error"] == nil {
		t.Error("expected error in output for empty cloud")
	}
}

func TestVectorCreateIndexStep_CaseInsensitiveCloud(t *testing.T) {
	setupMock(t)
	step := &internal.VectorCreateIndexStep{}
	cfg := map[string]any{
		"module":    "test-vs",
		"name":      "test-index",
		"dimension": 1536,
		"cloud":     "GCP",
		"region":    "us-central1",
	}

	result, err := step.Execute(context.Background(), nil, nil, nil, nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["created"] != "test-index" {
		t.Errorf("expected created=test-index, got %v", result.Output["created"])
	}
}

func TestProviderRegistry(t *testing.T) {
	mp := &mockProvider{}
	internal.RegisterProvider("test-reg", mp)
	defer internal.UnregisterProvider("test-reg")

	p, ok := internal.GetProvider("test-reg")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if p != mp {
		t.Error("expected same provider instance")
	}

	internal.UnregisterProvider("test-reg")
	_, ok = internal.GetProvider("test-reg")
	if ok {
		t.Error("expected provider to be removed")
	}
}
