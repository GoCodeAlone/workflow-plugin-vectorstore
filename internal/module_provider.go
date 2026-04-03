package internal

import (
	"context"
	"fmt"
)

// ProviderModule implements sdk.ModuleInstance for the vectorstore.provider module type.
// On Init it creates the appropriate adapter and registers it in the global registry.
type ProviderModule struct {
	name     string
	config   map[string]any
	provider VectorStoreProvider
}

func NewProviderModule(name string, config map[string]any) *ProviderModule {
	return &ProviderModule{name: name, config: config}
}

func (m *ProviderModule) Init() error {
	providerType, _ := m.config["provider"].(string)
	switch providerType {
	case "pinecone":
		m.provider = &PineconeAdapter{}
	case "milvus":
		m.provider = &MilvusAdapter{}
	default:
		return fmt.Errorf("vectorstore: unsupported provider %q (supported: pinecone, milvus)", providerType)
	}

	cfg := ProviderConfig{
		Provider:    providerType,
		APIKey:      stringFromMap(m.config, "api_key"),
		Environment: stringFromMap(m.config, "environment"),
		Host:        stringFromMap(m.config, "host"),
		Namespace:   stringFromMap(m.config, "namespace"),
		IndexName:   stringFromMap(m.config, "index_name"),
	}

	if err := m.provider.Connect(context.Background(), cfg); err != nil {
		return fmt.Errorf("vectorstore: connect: %w", err)
	}

	RegisterProvider(m.name, m.provider)
	return nil
}

func (m *ProviderModule) Start(_ context.Context) error { return nil }

func (m *ProviderModule) Stop(_ context.Context) error {
	UnregisterProvider(m.name)
	if m.provider != nil {
		return m.provider.Close()
	}
	return nil
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
