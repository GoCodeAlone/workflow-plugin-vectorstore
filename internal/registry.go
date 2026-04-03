package internal

import "sync"

var (
	providersMu sync.RWMutex
	providers    = map[string]VectorStoreProvider{}
)

// RegisterProvider stores a named provider for step lookups.
func RegisterProvider(name string, p VectorStoreProvider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = p
}

// GetProvider retrieves a named provider.
func GetProvider(name string) (VectorStoreProvider, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	p, ok := providers[name]
	return p, ok
}

// UnregisterProvider removes a named provider.
func UnregisterProvider(name string) {
	providersMu.Lock()
	defer providersMu.Unlock()
	delete(providers, name)
}
