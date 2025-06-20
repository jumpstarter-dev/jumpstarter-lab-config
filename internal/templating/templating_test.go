package templating

import (
	"os"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"
)

func TestApplyReplacements_Simple(t *testing.T) {
	data := "Hello $(var.name), welcome to $(param.place)!"
	replacements := map[string]string{
		"var.name":    "Alice",
		"param.place": "Wonderland",
	}
	expected := "Hello Alice, welcome to Wonderland!"
	result, err := applyReplacements(data, replacements)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyReplacements_WithSpaces(t *testing.T) {
	data := "User: $(   var.user   ), Location: $( param.location )"
	replacements := map[string]string{
		"var.user":       "Bob",
		"param.location": "Lab",
	}
	expected := "User: Bob, Location: Lab"
	result, err := applyReplacements(data, replacements)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyReplacements_MultipleOccurrences(t *testing.T) {
	data := "$(var.x) and $(var.x) again"
	replacements := map[string]string{
		"var.x": "42",
	}
	expected := "42 and 42 again"
	result, err := applyReplacements(data, replacements)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

}

func TestApplyReplacements_NoMatch(t *testing.T) {
	data := "Nothing to replace here either"
	replacements := map[string]string{
		"var.x": "42",
	}
	expected := data
	result, err := applyReplacements(data, replacements)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestApplyReplacements_EmptyReplacements(t *testing.T) {
	data := "Hello $(var.name)"
	replacements := map[string]string{}
	expected := data
	result, err := applyReplacements(data, replacements)
	if err == nil {
		t.Errorf("an error was expected for missing replacements, got nil")
	}

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
func TestProcessTemplate_Basic(t *testing.T) {
	input := "Hello $(var.name), welcome to $(param.place)!"
	expected := "Hello Alice, welcome to Wonderland!"
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = varsMock.Set("name", "Alice")

	params := &Parameters{
		parameters: map[string]string{"place": "Wonderland"},
	}

	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
func TestProcessTemplate_MultipleVariablesAndParams(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("user", "Charlie")
	_ = varsMock.Set("id", "007")

	params := &Parameters{
		parameters: map[string]string{"mission": "Secret", "location": "HQ"},
	}

	input := "Agent $(var.user) (#$(var.id)) on $(param.mission) at $(param.location)"
	expected := "Agent Charlie (#007) on Secret at HQ"

	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_EmptyParamsAndVars(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := &Parameters{
		parameters: map[string]string{},
	}
	input := "Nothing to replace here"
	expected := input
	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_MissingVariable_Error(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := &Parameters{
		parameters: map[string]string{},
	}
	input := "Hello $(var.missing)"
	_, err = ProcessTemplate(input, varsMock, params, nil)
	if err == nil {
		t.Errorf("expected error for missing variable, got nil")
	}
}

func TestProcessTemplate_ParameterOnly_NewVars(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := &Parameters{
		parameters: map[string]string{"foo": "bar"},
	}
	input := "Param: $(param.foo)"
	expected := "Param: bar"
	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
func TestProcessTemplate_NoReplacements(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("foo", "bar")
	params := &Parameters{
		parameters: map[string]string{"baz": "qux"},
	}
	input := "Nothing to replace here"
	expected := input
	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_MissingVariable(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := &Parameters{
		parameters: map[string]string{},
	}
	input := "Hello $(var.missing)"
	_, err = ProcessTemplate(input, varsMock, params, nil)
	if err == nil {
		t.Errorf("expected error for missing variable, got nil")
	}
}

func TestProcessTemplate_RecursiveReplacements(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("a", "$(var.b)")
	_ = varsMock.Set("b", "$(var.c)")
	_ = varsMock.Set("c", "42")

	params := &Parameters{
		parameters: map[string]string{},
	}

	input := "Value of a: $(var.a)"
	expected := "Value of a: 42"
	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
func TestProcessTemplate_RecursiveReplacements_Limit(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("a", "$(var.a)") // Recursive definition

	params := &Parameters{
		parameters: map[string]string{},
	}

	input := "Value of a: $(var.a)"
	expectedRecursionError := "templating: recursion limit reached while applying replacements, "
	expectedRecursionError += "check for circular references, like: var.a => $(var.a)"
	_, err = ProcessTemplate(input, varsMock, params, nil)
	if err == nil {
		t.Errorf("expected error for recursive replacement, got nil")
	} else if err.Error() != expectedRecursionError {
		t.Errorf("unexpected recursion limit error, got %v", err)
	}
}

// Test templating when introducing an ansible vault encrypted variable that can't be decrypted
func TestProcessTemplate_VaultDecryptionError(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Set a vault-encrypted variable that cannot be decrypted
	_ = varsMock.Set("vault_var", "$ANSIBLE_VAULT;1.1;AES256\n  6162636465666768696a6b6c6d6e6f70\n")

	params := &Parameters{
		parameters: map[string]string{},
	}

	// unset ANSIBLE_VAULT_PASSWORD_FILE
	if err := os.Unsetenv("ANSIBLE_VAULT_PASSWORD_FILE"); err != nil {
		t.Fatalf("failed to unset ANSIBLE_VAULT_PASSWORD_FILE: %v", err)
	}

	input := "Vault variable: $(var.vault_var)"
	_, err = ProcessTemplate(input, varsMock, params, nil)
	if err == nil {
		t.Errorf("Call should have failed, got nil")
	}
}

// Test structures for TemplateApplier tests
type TestStruct struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Tags        []string          `json:"tags"`
}

type NestedTestStruct struct {
	ID     string     `json:"id"`
	Config TestStruct `json:"config"`
}

func TestTemplateApplier_Apply_SimpleStruct(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("service", "web-server")
	_ = varsMock.Set("env", "production")

	params := NewParameters("test")
	params.Set("version", "1.0.0")

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	testObj := &TestStruct{
		Name:        "$(var.service)-$(param.version)",
		Description: "Running in $(var.env) environment",
		Labels: map[string]string{
			"service": "$(var.service)",
			"env":     "$(var.env)",
		},
		Tags: []string{"$(var.service)", "$(param.version)"},
	}

	err = applier.Apply(testObj)
	if err != nil {
		t.Fatalf("unexpected error applying templates: %v", err)
	}

	expected := &TestStruct{
		Name:        "web-server-1.0.0",
		Description: "Running in production environment",
		Labels: map[string]string{
			"service": "web-server",
			"env":     "production",
		},
		Tags: []string{"web-server", "1.0.0"},
	}

	if testObj.Name != expected.Name {
		t.Errorf("expected Name %q, got %q", expected.Name, testObj.Name)
	}
	if testObj.Description != expected.Description {
		t.Errorf("expected Description %q, got %q", expected.Description, testObj.Description)
	}
	if testObj.Labels["service"] != expected.Labels["service"] {
		t.Errorf("expected Labels[service] %q, got %q", expected.Labels["service"], testObj.Labels["service"])
	}
	if testObj.Labels["env"] != expected.Labels["env"] {
		t.Errorf("expected Labels[env] %q, got %q", expected.Labels["env"], testObj.Labels["env"])
	}
	if len(testObj.Tags) != len(expected.Tags) {
		t.Errorf("expected %d tags, got %d", len(expected.Tags), len(testObj.Tags))
	}
	for i, tag := range testObj.Tags {
		if tag != expected.Tags[i] {
			t.Errorf("expected Tags[%d] %q, got %q", i, expected.Tags[i], tag)
		}
	}
}

func TestTemplateApplier_Apply_NestedStruct(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("app", "my-app")

	params := NewParameters("test")
	params.Set("instance", "001")

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	testObj := &NestedTestStruct{
		ID: "$(var.app)-$(param.instance)",
		Config: TestStruct{
			Name:        "$(var.app)",
			Description: "Instance $(param.instance)",
			Labels: map[string]string{
				"app":      "$(var.app)",
				"instance": "$(param.instance)",
			},
		},
	}

	err = applier.Apply(testObj)
	if err != nil {
		t.Fatalf("unexpected error applying templates: %v", err)
	}

	if testObj.ID != "my-app-001" {
		t.Errorf("expected ID %q, got %q", "my-app-001", testObj.ID)
	}
	if testObj.Config.Name != "my-app" {
		t.Errorf("expected Config.Name %q, got %q", "my-app", testObj.Config.Name)
	}
	if testObj.Config.Description != "Instance 001" {
		t.Errorf("expected Config.Description %q, got %q", "Instance 001", testObj.Config.Description)
	}
	if testObj.Config.Labels["app"] != "my-app" {
		t.Errorf("expected Config.Labels[app] %q, got %q", "my-app", testObj.Config.Labels["app"])
	}
	if testObj.Config.Labels["instance"] != "001" {
		t.Errorf("expected Config.Labels[instance] %q, got %q", "001", testObj.Config.Labels["instance"])
	}
}

func TestTemplateApplier_Apply_NilPointer(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := NewParameters("test")
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	var testObj *TestStruct = nil
	err = applier.Apply(testObj)
	if err != nil {
		t.Fatalf("unexpected error applying templates to nil pointer: %v", err)
	}
}

func TestTemplateApplier_Apply_EmptySlice(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := NewParameters("test")
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	testObj := &TestStruct{
		Tags: []string{},
	}

	err = applier.Apply(testObj)
	if err != nil {
		t.Fatalf("unexpected error applying templates to empty slice: %v", err)
	}

	if len(testObj.Tags) != 0 {
		t.Errorf("expected empty slice, got %v", testObj.Tags)
	}
}

func TestTemplateApplier_Apply_MissingVariable(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := NewParameters("test")
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	testObj := &TestStruct{
		Name: "$(var.missing)",
	}

	err = applier.Apply(testObj)
	if err == nil {
		t.Errorf("expected error for missing variable, got nil")
	}
}

func TestTemplateApplier_NewTemplateApplier_NilConfig(t *testing.T) {
	params := NewParameters("test")
	_, err := NewTemplateApplier(nil, params)
	if err == nil || err.Error() != "config cannot be nil" {
		t.Errorf("expected 'config cannot be nil' error, got %v", err)
	}
}

func TestTemplateApplier_NewTemplateApplier_NilLoadedConfig(t *testing.T) {
	cfg := &config.Config{
		Loaded: nil,
	}
	params := NewParameters("test")
	_, err := NewTemplateApplier(cfg, params)
	if err == nil || err.Error() != "loaded config cannot be nil" {
		t.Errorf("expected 'loaded config cannot be nil' error, got %v", err)
	}
}

func TestProcessTemplate_WithMeta(t *testing.T) {
	input := "Hello $(var.name), welcome to $(param.place) this is $(someMeta)!"
	expected := "Hello Alice, welcome to Wonderland this is a meta variable!"
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = varsMock.Set("name", "Alice")

	params := &Parameters{
		parameters: map[string]string{"place": "Wonderland"},
	}

	meta := &Parameters{
		parameters: map[string]string{"someMeta": "a meta variable"},
	}

	result, err := ProcessTemplate(input, varsMock, params, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_NilMeta(t *testing.T) {
	input := "Hello $(var.name)"
	expected := "Hello Bob"
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = varsMock.Set("name", "Bob")

	params := &Parameters{
		parameters: map[string]string{},
	}

	result, err := ProcessTemplate(input, varsMock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_MissingMeta(t *testing.T) {
	input := "Meta: $(meta.missing)"
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := &Parameters{
		parameters: map[string]string{},
	}

	meta := &Parameters{
		parameters: map[string]string{},
	}

	_, err = ProcessTemplate(input, varsMock, params, meta)
	if err == nil {
		t.Errorf("expected error for missing meta parameter, got nil")
	}
}

func TestTemplateApplier_Apply_WithMeta(t *testing.T) {
	varsMock, err := vars.NewVariables("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = varsMock.Set("service", "web-server")

	params := NewParameters("test")
	params.Set("version", "1.0.0")

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			Variables: varsMock,
		},
	}

	applier, err := NewTemplateApplier(cfg, params)
	if err != nil {
		t.Fatalf("unexpected error creating applier: %v", err)
	}

	testObj := &TestStruct{
		Name:        "test",
		Description: "My name is $(name)",
		Labels: map[string]string{
			"service":    "$(var.service)",
			"version":    "$(param.version)",
			"env":        "$(name) somewhere",
			"datacenter": "$(name)'s datacenter",
		},
		Tags: []string{"$(var.service)", "$(name)"},
	}

	err = applier.Apply(testObj)
	if err != nil {
		t.Fatalf("unexpected error applying templates: %v", err)
	}

	expected := &TestStruct{
		Name:        "test",
		Description: "My name is test",
		Labels: map[string]string{
			"service":    "web-server",
			"version":    "1.0.0",
			"env":        "test somewhere",
			"datacenter": "test's datacenter",
		},
		Tags: []string{"web-server", "test"},
	}

	if testObj.Name != expected.Name {
		t.Errorf("expected Name %q, got %q", expected.Name, testObj.Name)
	}
	if testObj.Description != expected.Description {
		t.Errorf("expected Description %q, got %q", expected.Description, testObj.Description)
	}
	if testObj.Labels["service"] != expected.Labels["service"] {
		t.Errorf("expected Labels[service] %q, got %q", expected.Labels["service"], testObj.Labels["service"])
	}
	if testObj.Labels["version"] != expected.Labels["version"] {
		t.Errorf("expected Labels[version] %q, got %q", expected.Labels["version"], testObj.Labels["version"])
	}
	if testObj.Labels["env"] != expected.Labels["env"] {
		t.Errorf("expected Labels[env] %q, got %q", expected.Labels["env"], testObj.Labels["env"])
	}
	if testObj.Labels["datacenter"] != expected.Labels["datacenter"] {
		t.Errorf("expected Labels[datacenter] %q, got %q", expected.Labels["datacenter"], testObj.Labels["datacenter"])
	}
	if len(testObj.Tags) != len(expected.Tags) {
		t.Errorf("expected %d tags, got %d", len(expected.Tags), len(testObj.Tags))
	}
	for i, tag := range testObj.Tags {
		if tag != expected.Tags[i] {
			t.Errorf("expected Tags[%d] %q, got %q", i, expected.Tags[i], tag)
		}
	}
}
