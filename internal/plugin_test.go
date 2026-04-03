package internal_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-vectorstore/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestNewPlugin_ImplementsPluginProvider(t *testing.T) {
	var _ sdk.PluginProvider = internal.NewPlugin()
}

func TestNewPlugin_ImplementsModuleProvider(t *testing.T) {
	p := internal.NewPlugin()
	mp, ok := p.(interface {
		ModuleTypes() []string
		CreateModule(string, string, map[string]any) (sdk.ModuleInstance, error)
	})
	if !ok {
		t.Fatal("plugin does not implement ModuleProvider")
	}
	types := mp.ModuleTypes()
	if len(types) != 1 || types[0] != "vectorstore.provider" {
		t.Errorf("unexpected module types: %v", types)
	}
}

func TestNewPlugin_ImplementsStepProvider(t *testing.T) {
	p := internal.NewPlugin()
	sp, ok := p.(interface {
		StepTypes() []string
		CreateStep(string, string, map[string]any) (sdk.StepInstance, error)
	})
	if !ok {
		t.Fatal("plugin does not implement StepProvider")
	}
	types := sp.StepTypes()
	expected := []string{
		"step.vector_upsert",
		"step.vector_query",
		"step.vector_fetch",
		"step.vector_delete",
		"step.vector_create_index",
		"step.vector_list_indexes",
		"step.vector_describe_index",
	}
	if len(types) != len(expected) {
		t.Fatalf("expected %d step types, got %d", len(expected), len(types))
	}
	for i, e := range expected {
		if types[i] != e {
			t.Errorf("step type %d: expected %q, got %q", i, e, types[i])
		}
	}
}

func TestManifest_HasRequiredFields(t *testing.T) {
	m := internal.Manifest
	if m.Name == "" {
		t.Error("manifest Name is empty")
	}
	if m.Version == "" {
		t.Error("manifest Version is empty")
	}
	if m.Description == "" {
		t.Error("manifest Description is empty")
	}
}

func TestCreateStep_ReturnsInstances(t *testing.T) {
	p := internal.NewPlugin()
	sp := p.(interface {
		CreateStep(string, string, map[string]any) (sdk.StepInstance, error)
	})

	stepTypes := []string{
		"step.vector_upsert",
		"step.vector_query",
		"step.vector_fetch",
		"step.vector_delete",
		"step.vector_create_index",
		"step.vector_list_indexes",
		"step.vector_describe_index",
	}

	for _, st := range stepTypes {
		step, err := sp.CreateStep(st, "test", map[string]any{"module": "test"})
		if err != nil {
			t.Errorf("CreateStep(%q): unexpected error: %v", st, err)
			continue
		}
		if step == nil {
			t.Errorf("CreateStep(%q): returned nil", st)
		}
	}
}

func TestCreateStep_UnknownType(t *testing.T) {
	p := internal.NewPlugin()
	sp := p.(interface {
		CreateStep(string, string, map[string]any) (sdk.StepInstance, error)
	})

	_, err := sp.CreateStep("step.nonexistent", "test", nil)
	if err == nil {
		t.Error("expected error for unknown step type")
	}
}

func TestCreateModule_UnknownType(t *testing.T) {
	p := internal.NewPlugin()
	mp := p.(interface {
		CreateModule(string, string, map[string]any) (sdk.ModuleInstance, error)
	})

	_, err := mp.CreateModule("nonexistent", "test", nil)
	if err == nil {
		t.Error("expected error for unknown module type")
	}
}
