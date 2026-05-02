package internal

import (
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Version is set at build time via -ldflags
// "-X github.com/GoCodeAlone/workflow-plugin-vectorstore/internal.Version=X.Y.Z"
var Version = "dev"

// Manifest holds the plugin metadata used by the workflow engine for
// discovery and capability negotiation.
var Manifest = sdk.PluginManifest{
	Name:        "workflow-plugin-vectorstore",
	Version:     Version,
	Author:      "GoCodeAlone",
	Description: "Vector database integration (Pinecone, Milvus) for RAG pipelines",
}

type plugin struct{}

// NewPlugin creates a new plugin instance implementing PluginProvider,
// ModuleProvider, StepProvider, and SchemaProvider. SchemaProvider exposes
// module config schemas over gRPC so host engines and tooling (MCP, LSP,
// wfctl) can provide config guidance without loading the plugin binary.
func NewPlugin() sdk.PluginProvider {
	return &plugin{}
}

func (p *plugin) Manifest() sdk.PluginManifest {
	return Manifest
}

// --- ModuleProvider ---

func (p *plugin) ModuleTypes() []string {
	return []string{
		"vectorstore.provider",
	}
}

func (p *plugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "vectorstore.provider":
		return NewProviderModule(name, config), nil
	default:
		return nil, fmt.Errorf("unknown module type %q", typeName)
	}
}

// --- StepProvider ---

func (p *plugin) StepTypes() []string {
	return []string{
		"step.vector_upsert",
		"step.vector_query",
		"step.vector_fetch",
		"step.vector_delete",
		"step.vector_create_index",
		"step.vector_list_indexes",
		"step.vector_describe_index",
	}
}

func (p *plugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.vector_upsert":
		return &VectorUpsertStep{config: config}, nil
	case "step.vector_query":
		return &VectorQueryStep{config: config}, nil
	case "step.vector_fetch":
		return &VectorFetchStep{config: config}, nil
	case "step.vector_delete":
		return &VectorDeleteStep{config: config}, nil
	case "step.vector_create_index":
		return &VectorCreateIndexStep{config: config}, nil
	case "step.vector_list_indexes":
		return &VectorListIndexesStep{config: config}, nil
	case "step.vector_describe_index":
		return &VectorDescribeIndexStep{config: config}, nil
	default:
		return nil, fmt.Errorf("unknown step type %q", typeName)
	}
}

// --- SchemaProvider ---

// ModuleSchemas returns UI/tooling schema descriptors for each module type
// this plugin advertises. These are served over gRPC so the host engine can
// present config guidance without requiring the plugin to be loaded.
func (p *plugin) ModuleSchemas() []sdk.ModuleSchemaData {
	return []sdk.ModuleSchemaData{
		{
			Type:        "vectorstore.provider",
			Label:       "Vector Store Provider",
			Category:    "Vector Database",
			Description: "Initializes a Pinecone adapter and registers it by module name for use by vector step types. Milvus support is planned but not yet implemented.",
			ConfigFields: []sdk.ConfigField{
				{
					Name:        "provider",
					Type:        "string",
					Description: "Backend provider. Currently only 'pinecone' is supported (milvus is planned but not yet implemented).",
					Required:    true,
					Options:     []string{"pinecone"},
				},
				{
					Name:        "api_key",
					Type:        "string",
					Description: "API key for authenticating with the vector database service.",
					Required:    true,
				},
				{
					Name:        "environment",
					Type:        "string",
					Description: "Environment name (used by legacy Pinecone environments).",
					Required:    false,
				},
				{
					Name:        "host",
					Type:        "string",
					Description: "Custom host URL for self-hosted or private-link endpoints.",
					Required:    false,
				},
				{
					Name:        "namespace",
					Type:        "string",
					Description: "Default namespace for all vector operations in this module instance.",
					Required:    false,
				},
				{
					Name:        "index_name",
					Type:        "string",
					Description: "Default index name for operations that do not specify one explicitly.",
					Required:    false,
				},
			},
		},
	}
}
