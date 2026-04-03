# CLAUDE.md — workflow-plugin-vectorstore

Vector database integration plugin (Pinecone, Milvus) for RAG pipelines. External gRPC plugin for the GoCodeAlone/workflow engine.

## Build & Test

```sh
go build ./...
go test ./... -v -race -count=1
```

## Cross-compile for deployment

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o workflow-plugin-vectorstore ./cmd/workflow-plugin-vectorstore/
```

## Structure

- `cmd/workflow-plugin-vectorstore/main.go` — Plugin entry point (calls `sdk.Serve`)
- `internal/plugin.go` — PluginProvider + ModuleProvider + StepProvider
- `internal/vectorstore.go` — VectorStoreProvider interface and domain types
- `internal/registry.go` — Global provider registry (module name → adapter)
- `internal/pinecone_adapter.go` — Pinecone adapter (go-pinecone/v5)
- `internal/milvus_adapter.go` — Milvus adapter (stub, TODO)
- `internal/module_provider.go` — `vectorstore.provider` module implementation
- `internal/steps.go` — All 7 step type implementations
- `internal/helpers.go` — Config parsing utilities
- `plugin.json` — Capability manifest for the workflow registry

## Module

- `vectorstore.provider` — Initializes a Pinecone or Milvus adapter and registers it by module name.

## Step Types

- `step.vector_upsert` — Upsert vectors with IDs, values, and metadata
- `step.vector_query` — Similarity search returning scored matches
- `step.vector_fetch` — Fetch vectors by ID
- `step.vector_delete` — Delete by IDs or metadata filter
- `step.vector_create_index` — Create a new vector index
- `step.vector_list_indexes` — List all indexes
- `step.vector_describe_index` — Get index info (dimension, metric, count)

## Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```
GoReleaser builds cross-platform binaries and creates a GitHub Release automatically.
