package internal

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func getProvider(config map[string]any) (VectorStoreProvider, error) {
	moduleName, _ := config["module"].(string)
	if moduleName == "" {
		return nil, fmt.Errorf("vectorstore step: 'module' config is required")
	}
	p, ok := GetProvider(moduleName)
	if !ok {
		return nil, fmt.Errorf("vectorstore step: provider %q not found", moduleName)
	}
	return p, nil
}

// --- step.vector_upsert ---

type VectorUpsertStep struct{ config map[string]any }

func (s *VectorUpsertStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	rawVectors, _ := merged["vectors"].([]any)
	vectors, err := parseVectors(rawVectors)
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return &sdk.StepResult{Output: map[string]any{"error": "vector_upsert: 'vectors' is required and must be non-empty"}}, nil
	}

	if err := p.Upsert(ctx, vectors); err != nil {
		return nil, fmt.Errorf("vector_upsert: %w", err)
	}

	return &sdk.StepResult{Output: map[string]any{
		"upserted_count": len(vectors),
	}}, nil
}

// --- step.vector_query ---

type VectorQueryStep struct{ config map[string]any }

func (s *VectorQueryStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	vector, err := parseFloat32Slice(merged["vector"])
	if err != nil {
		return nil, fmt.Errorf("vector_query: %w", err)
	}

	topK := intFromMap(merged, "top_k", 10)
	if topK <= 0 {
		return &sdk.StepResult{Output: map[string]any{"error": "vector_query: 'top_k' must be a positive integer"}}, nil
	}
	filter, _ := merged["filter"].(map[string]any)

	matches, err := p.Query(ctx, vector, topK, filter)
	if err != nil {
		return nil, fmt.Errorf("vector_query: %w", err)
	}

	results := make([]any, len(matches))
	for i, m := range matches {
		results[i] = map[string]any{
			"id":       m.ID,
			"score":    m.Score,
			"metadata": m.Metadata,
			"values":   m.Values,
		}
	}

	return &sdk.StepResult{Output: map[string]any{
		"matches": results,
		"count":   len(matches),
	}}, nil
}

// --- step.vector_fetch ---

type VectorFetchStep struct{ config map[string]any }

func (s *VectorFetchStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	ids := parseStringSlice(merged["ids"])
	if len(ids) == 0 {
		return nil, fmt.Errorf("vector_fetch: 'ids' is required")
	}

	vectors, err := p.Fetch(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("vector_fetch: %w", err)
	}

	results := make([]any, len(vectors))
	for i, v := range vectors {
		results[i] = map[string]any{
			"id":       v.ID,
			"values":   v.Values,
			"metadata": v.Metadata,
		}
	}

	return &sdk.StepResult{Output: map[string]any{
		"vectors": results,
		"count":   len(vectors),
	}}, nil
}

// --- step.vector_delete ---

type VectorDeleteStep struct{ config map[string]any }

func (s *VectorDeleteStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	ids := parseStringSlice(merged["ids"])
	filter, _ := merged["filter"].(map[string]any)

	if len(ids) > 0 && filter != nil {
		return nil, fmt.Errorf("vector_delete: 'ids' and 'filter' are mutually exclusive — provide exactly one")
	}

	if len(ids) > 0 {
		if err := p.Delete(ctx, ids); err != nil {
			return nil, fmt.Errorf("vector_delete: %w", err)
		}
		return &sdk.StepResult{Output: map[string]any{"deleted_count": len(ids)}}, nil
	}

	if len(filter) > 0 {
		if err := p.DeleteByFilter(ctx, filter); err != nil {
			return nil, fmt.Errorf("vector_delete: %w", err)
		}
		return &sdk.StepResult{Output: map[string]any{"deleted_by_filter": true}}, nil
	}

	return nil, fmt.Errorf("vector_delete: either 'ids' or 'filter' is required")
}

// --- step.vector_create_index ---

type VectorCreateIndexStep struct{ config map[string]any }

func (s *VectorCreateIndexStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	cfg := IndexConfig{
		Name:      stringFromMap(merged, "name"),
		Dimension: intFromMap(merged, "dimension", 0),
		Metric:    stringFromMap(merged, "metric"),
		Cloud:     stringFromMap(merged, "cloud"),
		Region:    stringFromMap(merged, "region"),
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("vector_create_index: 'name' is required")
	}
	if cfg.Dimension == 0 {
		return nil, fmt.Errorf("vector_create_index: 'dimension' is required")
	}

	switch strings.ToLower(cfg.Cloud) {
	case "aws", "gcp", "azure":
		cfg.Cloud = strings.ToLower(cfg.Cloud)
	default:
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("vector_create_index: 'cloud' must be one of aws, gcp, azure; got %q", cfg.Cloud)}}, nil
	}

	if err := p.CreateIndex(ctx, cfg); err != nil {
		return nil, fmt.Errorf("vector_create_index: %w", err)
	}

	return &sdk.StepResult{Output: map[string]any{
		"created": cfg.Name,
	}}, nil
}

// --- step.vector_list_indexes ---

type VectorListIndexesStep struct{ config map[string]any }

func (s *VectorListIndexesStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	indexes, err := p.ListIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("vector_list_indexes: %w", err)
	}

	results := make([]any, len(indexes))
	for i, idx := range indexes {
		results[i] = map[string]any{
			"name":         idx.Name,
			"dimension":    idx.Dimension,
			"metric":       idx.Metric,
			"vector_count": idx.VectorCount,
		}
	}

	return &sdk.StepResult{Output: map[string]any{
		"indexes": results,
		"count":   len(indexes),
	}}, nil
}

// --- step.vector_describe_index ---

type VectorDescribeIndexStep struct{ config map[string]any }

func (s *VectorDescribeIndexStep) Execute(ctx context.Context, triggerData map[string]any, stepOutputs map[string]map[string]any, current, metadata, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	p, err := getProvider(merged)
	if err != nil {
		return nil, err
	}

	name := stringFromMap(merged, "name")
	if name == "" {
		return nil, fmt.Errorf("vector_describe_index: 'name' is required")
	}

	info, err := p.DescribeIndex(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("vector_describe_index: %w", err)
	}

	return &sdk.StepResult{Output: map[string]any{
		"name":         info.Name,
		"dimension":    info.Dimension,
		"metric":       info.Metric,
		"vector_count": info.VectorCount,
	}}, nil
}
