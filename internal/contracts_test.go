package internal_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-vectorstore/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// TestSchemaProvider_ImplementedByPlugin verifies that the plugin implements
// the sdk.SchemaProvider interface, which exposes module schema descriptors
// over gRPC for tooling (MCP, LSP, wfctl).
func TestSchemaProvider_ImplementedByPlugin(t *testing.T) {
	p := internal.NewPlugin()
	sp, ok := p.(sdk.SchemaProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.SchemaProvider")
	}

	schemas := sp.ModuleSchemas()
	if len(schemas) == 0 {
		t.Fatal("ModuleSchemas() returned no schemas")
	}
}

// TestSchemaProvider_CoversAllModuleTypes verifies that every module type
// advertised by the plugin has a corresponding schema descriptor.
func TestSchemaProvider_CoversAllModuleTypes(t *testing.T) {
	p := internal.NewPlugin()

	mp := p.(interface {
		ModuleTypes() []string
	})
	sp := p.(sdk.SchemaProvider)

	moduleTypes := mp.ModuleTypes()
	schemas := sp.ModuleSchemas()

	schemaByType := make(map[string]struct{}, len(schemas))
	for _, s := range schemas {
		schemaByType[s.Type] = struct{}{}
	}

	for _, mt := range moduleTypes {
		if _, ok := schemaByType[mt]; !ok {
			t.Errorf("module type %q has no schema descriptor", mt)
		}
	}
}

// TestSchemaProvider_VectorstoreProviderSchema verifies the vectorstore.provider
// module schema has the required config fields.
func TestSchemaProvider_VectorstoreProviderSchema(t *testing.T) {
	p := internal.NewPlugin()
	sp := p.(sdk.SchemaProvider)

	var providerSchema *sdk.ModuleSchemaData
	for _, s := range sp.ModuleSchemas() {
		if s.Type == "vectorstore.provider" {
			s := s // copy
			providerSchema = &s
			break
		}
	}
	if providerSchema == nil {
		t.Fatal("no schema found for vectorstore.provider")
	}

	if providerSchema.Description == "" {
		t.Error("vectorstore.provider schema has empty description")
	}

	requiredFields := map[string]bool{"provider": false, "api_key": false}
	for _, cf := range providerSchema.ConfigFields {
		if _, ok := requiredFields[cf.Name]; ok {
			requiredFields[cf.Name] = true
			if !cf.Required {
				t.Errorf("config field %q should be marked required", cf.Name)
			}
		}
	}
	for field, found := range requiredFields {
		if !found {
			t.Errorf("required config field %q not found in vectorstore.provider schema", field)
		}
	}
}

// contractsFile is the layout of plugin.contracts.json used for strict contract auditing.
type contractsFile struct {
	Version string                       `json:"version"`
	Plugin  string                       `json:"plugin"`
	Modules map[string]contractDescriptor `json:"modules"`
	Steps   map[string]contractDescriptor `json:"steps"`
}

type contractDescriptor struct {
	Description    string                 `json:"description"`
	RequiredInputs map[string]fieldSpec   `json:"required_inputs"`
	OptionalInputs map[string]fieldSpec   `json:"optional_inputs"`
	Outputs        map[string]fieldSpec   `json:"outputs"`
}

type fieldSpec struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// TestContractsFile_ExistsAndValid verifies plugin.contracts.json is present,
// parseable, and covers the expected module and step types.
func TestContractsFile_ExistsAndValid(t *testing.T) {
	data, err := os.ReadFile("../plugin.contracts.json")
	if err != nil {
		t.Fatalf("plugin.contracts.json not found: %v", err)
	}

	var cf contractsFile
	if err := json.Unmarshal(data, &cf); err != nil {
		t.Fatalf("plugin.contracts.json is not valid JSON: %v", err)
	}

	if cf.Plugin == "" {
		t.Error("plugin.contracts.json: 'plugin' field is empty")
	}
	if cf.Version == "" {
		t.Error("plugin.contracts.json: 'version' field is empty")
	}
}

// TestContractsFile_CoversAllModuleTypes verifies that plugin.contracts.json
// contains a contract descriptor for every advertised module type.
func TestContractsFile_CoversAllModuleTypes(t *testing.T) {
	data, err := os.ReadFile("../plugin.contracts.json")
	if err != nil {
		t.Fatalf("plugin.contracts.json not found: %v", err)
	}
	var cf contractsFile
	if err := json.Unmarshal(data, &cf); err != nil {
		t.Fatalf("plugin.contracts.json parse: %v", err)
	}

	p := internal.NewPlugin()
	mp := p.(interface{ ModuleTypes() []string })

	for _, mt := range mp.ModuleTypes() {
		desc, ok := cf.Modules[mt]
		if !ok {
			t.Errorf("missing_module_contract_descriptor: no entry in plugin.contracts.json for module type %q", mt)
			continue
		}
		if desc.Description == "" {
			t.Errorf("module contract for %q has empty description", mt)
		}
	}
}

// TestContractsFile_CoversAllStepTypes verifies that plugin.contracts.json
// contains a contract descriptor for every advertised step type.
func TestContractsFile_CoversAllStepTypes(t *testing.T) {
	data, err := os.ReadFile("../plugin.contracts.json")
	if err != nil {
		t.Fatalf("plugin.contracts.json not found: %v", err)
	}
	var cf contractsFile
	if err := json.Unmarshal(data, &cf); err != nil {
		t.Fatalf("plugin.contracts.json parse: %v", err)
	}

	p := internal.NewPlugin()
	sp := p.(interface{ StepTypes() []string })

	for _, st := range sp.StepTypes() {
		desc, ok := cf.Steps[st]
		if !ok {
			t.Errorf("missing_step_contract_descriptor: no entry in plugin.contracts.json for step type %q", st)
			continue
		}
		if desc.Description == "" {
			t.Errorf("step contract for %q has empty description", st)
		}
		if len(desc.RequiredInputs) == 0 && len(desc.OptionalInputs) == 0 {
			t.Errorf("step contract for %q has no defined inputs", st)
		}
	}
}

// TestPluginJSON_ContainsStepSchemas verifies that plugin.json includes
// stepSchemas for all advertised step types (strict step contract descriptors).
func TestPluginJSON_ContainsStepSchemas(t *testing.T) {
	data, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatalf("plugin.json not found: %v", err)
	}

	var manifest struct {
		StepSchemas []struct {
			Type string `json:"type"`
		} `json:"stepSchemas"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("plugin.json parse: %v", err)
	}

	if len(manifest.StepSchemas) == 0 {
		t.Fatal("plugin.json: stepSchemas is empty — strict step contract descriptors are missing")
	}

	p := internal.NewPlugin()
	sp := p.(interface{ StepTypes() []string })

	schemaByType := make(map[string]struct{}, len(manifest.StepSchemas))
	for _, s := range manifest.StepSchemas {
		schemaByType[s.Type] = struct{}{}
	}

	for _, st := range sp.StepTypes() {
		if _, ok := schemaByType[st]; !ok {
			t.Errorf("plugin.json stepSchemas missing strict contract for step type %q", st)
		}
	}
}
