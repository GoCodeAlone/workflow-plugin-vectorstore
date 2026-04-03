package internal

import "fmt"

func mergeConfig(base, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	return merged
}

func intFromMap(m map[string]any, key string, def int) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return def
}

func parseStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func parseFloat32Slice(v any) ([]float32, error) {
	switch val := v.(type) {
	case []float32:
		return val, nil
	case []any:
		out := make([]float32, 0, len(val))
		for _, item := range val {
			switch n := item.(type) {
			case float64:
				out = append(out, float32(n))
			case float32:
				out = append(out, n)
			default:
				return nil, fmt.Errorf("expected numeric value, got %T", item)
			}
		}
		return out, nil
	case []float64:
		out := make([]float32, len(val))
		for i, n := range val {
			out[i] = float32(n)
		}
		return out, nil
	}
	return nil, fmt.Errorf("'vector' must be a numeric array")
}

func parseVectors(raw []any) ([]Vector, error) {
	vectors := make([]Vector, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each vector must be a map with id, values, metadata")
		}
		id, _ := m["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("vector 'id' is required")
		}
		vals, err := parseFloat32Slice(m["values"])
		if err != nil {
			return nil, fmt.Errorf("vector %q values: %w", id, err)
		}
		md, _ := m["metadata"].(map[string]any)
		vectors = append(vectors, Vector{ID: id, Values: vals, Metadata: md})
	}
	return vectors, nil
}
