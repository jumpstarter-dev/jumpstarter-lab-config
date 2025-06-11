package templating

import (
	"testing"

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

	result, err := ProcessTemplate(input, varsMock, params)
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

	result, err := ProcessTemplate(input, varsMock, params)
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
	result, err := ProcessTemplate(input, varsMock, params)
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
	_, err = ProcessTemplate(input, varsMock, params)
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
	result, err := ProcessTemplate(input, varsMock, params)
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
	result, err := ProcessTemplate(input, varsMock, params)
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
	_, err = ProcessTemplate(input, varsMock, params)
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
	result, err := ProcessTemplate(input, varsMock, params)
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
	_, err = ProcessTemplate(input, varsMock, params)
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

	input := "Vault variable: $(var.vault_var)"
	_, err = ProcessTemplate(input, varsMock, params)
	if err == nil || err.Error() != "templating: error retrieving variable 'vault_var': "+
		"ANSIBLE_VAULT_PASSWORD_FILE or ANSIBLE_VAULT_PASSWORD required for encrypted key vault_var" {
		t.Errorf("unexpected error message, got %v", err)
	}
}
