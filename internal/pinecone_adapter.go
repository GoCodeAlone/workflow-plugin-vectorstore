package internal

import (
	"context"
	"fmt"

	"github.com/pinecone-io/go-pinecone/v5/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

// PineconeAdapter implements VectorStoreProvider using the Pinecone SDK.
type PineconeAdapter struct {
	client *pinecone.Client
	idx    *pinecone.IndexConnection
}

var _ VectorStoreProvider = (*PineconeAdapter)(nil)

func (a *PineconeAdapter) Connect(ctx context.Context, cfg ProviderConfig) error {
	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: cfg.APIKey,
		Host:   cfg.Host,
	})
	if err != nil {
		return fmt.Errorf("pinecone: new client: %w", err)
	}
	a.client = client

	if cfg.IndexName == "" {
		return nil
	}

	desc, err := client.DescribeIndex(ctx, cfg.IndexName)
	if err != nil {
		return fmt.Errorf("pinecone: describe index %q: %w", cfg.IndexName, err)
	}

	connParams := pinecone.NewIndexConnParams{
		Host:      desc.Host,
		Namespace: cfg.Namespace,
	}
	idxConn, err := client.Index(connParams)
	if err != nil {
		return fmt.Errorf("pinecone: index connection: %w", err)
	}
	a.idx = idxConn
	return nil
}

func (a *PineconeAdapter) Upsert(ctx context.Context, vectors []Vector) error {
	if a.idx == nil {
		return fmt.Errorf("pinecone: not connected to an index")
	}

	pvecs := make([]*pinecone.Vector, len(vectors))
	for i, v := range vectors {
		vals := make([]float32, len(v.Values))
		copy(vals, v.Values)

		pv := &pinecone.Vector{
			Id:     v.ID,
			Values: &vals,
		}
		if len(v.Metadata) > 0 {
			md, err := structpb.NewStruct(v.Metadata)
			if err != nil {
				return fmt.Errorf("pinecone: metadata for %q: %w", v.ID, err)
			}
			pv.Metadata = md
		}
		pvecs[i] = pv
	}

	_, err := a.idx.UpsertVectors(ctx, pvecs)
	return err
}

func (a *PineconeAdapter) Query(ctx context.Context, vector []float32, topK int, filter map[string]any) ([]Match, error) {
	if a.idx == nil {
		return nil, fmt.Errorf("pinecone: not connected to an index")
	}

	req := &pinecone.QueryByVectorValuesRequest{
		Vector:          vector,
		TopK:            uint32(topK),
		IncludeMetadata: true,
		IncludeValues:   true,
	}

	if len(filter) > 0 {
		mf, err := structpb.NewStruct(filter)
		if err != nil {
			return nil, fmt.Errorf("pinecone: filter: %w", err)
		}
		req.MetadataFilter = mf
	}

	resp, err := a.idx.QueryByVectorValues(ctx, req)
	if err != nil {
		return nil, err
	}

	matches := make([]Match, 0, len(resp.Matches))
	for _, sv := range resp.Matches {
		m := Match{
			Score: sv.Score,
		}
		if sv.Vector != nil {
			m.ID = sv.Vector.Id
			if sv.Vector.Values != nil {
				m.Values = *sv.Vector.Values
			}
			if sv.Vector.Metadata != nil {
				m.Metadata = sv.Vector.Metadata.AsMap()
			}
		}
		matches = append(matches, m)
	}
	return matches, nil
}

func (a *PineconeAdapter) Fetch(ctx context.Context, ids []string) ([]Vector, error) {
	if a.idx == nil {
		return nil, fmt.Errorf("pinecone: not connected to an index")
	}

	resp, err := a.idx.FetchVectors(ctx, ids)
	if err != nil {
		return nil, err
	}

	vectors := make([]Vector, 0, len(resp.Vectors))
	for _, pv := range resp.Vectors {
		v := Vector{ID: pv.Id}
		if pv.Values != nil {
			v.Values = *pv.Values
		}
		if pv.Metadata != nil {
			v.Metadata = pv.Metadata.AsMap()
		}
		vectors = append(vectors, v)
	}
	return vectors, nil
}

func (a *PineconeAdapter) Delete(ctx context.Context, ids []string) error {
	if a.idx == nil {
		return fmt.Errorf("pinecone: not connected to an index")
	}
	return a.idx.DeleteVectorsById(ctx, ids)
}

func (a *PineconeAdapter) DeleteByFilter(ctx context.Context, filter map[string]any) error {
	if a.idx == nil {
		return fmt.Errorf("pinecone: not connected to an index")
	}
	mf, err := structpb.NewStruct(filter)
	if err != nil {
		return fmt.Errorf("pinecone: filter: %w", err)
	}
	return a.idx.DeleteVectorsByFilter(ctx, mf)
}

func (a *PineconeAdapter) CreateIndex(ctx context.Context, cfg IndexConfig) error {
	if a.client == nil {
		return fmt.Errorf("pinecone: client not initialized")
	}

	dim := int32(cfg.Dimension)
	cloud := pinecone.Aws
	switch cfg.Cloud {
	case "gcp":
		cloud = pinecone.Gcp
	case "azure":
		cloud = pinecone.Azure
	}

	req := &pinecone.CreateServerlessIndexRequest{
		Name:      cfg.Name,
		Dimension: &dim,
		Cloud:     cloud,
		Region:    cfg.Region,
	}

	if cfg.Metric != "" {
		metric := pinecone.IndexMetric(cfg.Metric)
		req.Metric = &metric
	}

	_, err := a.client.CreateServerlessIndex(ctx, req)
	return err
}

func (a *PineconeAdapter) ListIndexes(ctx context.Context) ([]IndexInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("pinecone: client not initialized")
	}

	indexes, err := a.client.ListIndexes(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]IndexInfo, 0, len(indexes))
	for _, idx := range indexes {
		info := IndexInfo{
			Name:   idx.Name,
			Metric: string(idx.Metric),
		}
		if idx.Dimension != nil {
			info.Dimension = int(*idx.Dimension)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (a *PineconeAdapter) DescribeIndex(ctx context.Context, name string) (*IndexInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("pinecone: client not initialized")
	}

	idx, err := a.client.DescribeIndex(ctx, name)
	if err != nil {
		return nil, err
	}

	info := &IndexInfo{
		Name:   idx.Name,
		Metric: string(idx.Metric),
	}
	if idx.Dimension != nil {
		info.Dimension = int(*idx.Dimension)
	}
	return info, nil
}

func (a *PineconeAdapter) Close() error {
	if a.idx != nil {
		return a.idx.Close()
	}
	return nil
}
