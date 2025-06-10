package vars

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_vars.yaml")

	testContent := `simple_var: "hello world"
number_var: 42
vault_var: !vault |
          $ANSIBLE_VAULT;1.1;AES256
          64396432643133643937353139613831356532653834383533646462326466653839663866663933
          6461336163333733393032613632623364343162363737330a643939356364316132616236376165
          65636634643637653233383339663337383065613666313835333731373466666432666536396234
          6433396531666363370a663234646164656165343735653334306238326137663464323033623733
          3532
bool_var: true
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	vars, err := NewVariables("")
	if err != nil {
		t.Fatalf("Failed to create Variables instance: %v", err)
	}

	// Test successful loading
	err = vars.LoadFromFile(testFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if vars == nil {
		t.Fatal("LoadFromFile returned nil Variables")
	}

	if vars.data == nil {
		t.Fatal("Variables data is nil")
	}
}

func TestLoadFromFileErrors(t *testing.T) {

	vars, err := NewVariables("")
	if err != nil {
		t.Fatalf("Failed to create Variables instance: %v", err)
	}

	// Test non-existent file
	err = vars.LoadFromFile("non_existent_file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid YAML
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	invalidContent := `invalid: yaml: content:
  - missing
    proper: indentation
`

	err = os.WriteFile(invalidFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid test file: %v", err)
	}

	err = vars.LoadFromFile(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestGet(t *testing.T) {
	vars := &Variables{
		data: map[string]interface{}{
			"string_var": "test_value",
			"number_var": 42,
			"bool_var":   true,
		},
	}

	// Test string value
	value, err := vars.Get("string_var")
	if err != nil {
		t.Error("Expected string key to exist")
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %s", value)
	}

	// Test non-string value
	_, err = vars.Get("number_var")
	if err == nil {
		t.Error("Expected failure for non-string key")
	}

	// Test non-existing key
	_, err = vars.Get("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestGetAllKeys(t *testing.T) {
	vars := &Variables{
		data: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	keys := vars.GetAllKeys()

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check that all expected keys are present
	expectedKeys := map[string]bool{"key1": false, "key2": false, "key3": false}
	for _, key := range keys {
		if _, exists := expectedKeys[key]; exists {
			expectedKeys[key] = true
		} else {
			t.Errorf("Unexpected key: %s", key)
		}
	}

	// Check that all expected keys were found
	for key, found := range expectedKeys {
		if !found {
			t.Errorf("Expected key not found: %s", key)
		}
	}
}

func TestHas(t *testing.T) {
	vars := &Variables{
		data: map[string]interface{}{
			"existing_key": "value",
		},
	}

	// Test existing key
	if !vars.Has("existing_key") {
		t.Error("Expected existing_key to exist")
	}

	// Test non-existing key
	if vars.Has("non_existent") {
		t.Error("Expected non_existent to not exist")
	}
}

func TestIntegrationWithExampleFile(t *testing.T) {
	// Test with the actual example file structure
	// first acquite the path to this test file
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	exampleFile := filepath.Join(currentDir, "..", "..", "example", "vars.yaml")
	vars, err := NewVariables("")
	if err != nil {
		t.Fatalf("Failed to create Variables instance: %v", err)
	}

	err = vars.LoadFromFile(exampleFile)
	if err != nil {
		t.Fatalf("Failed to load example file: %v", err)
	}

	// Test ti-exporter-image
	image, err := vars.Get("ti-exporter-image")
	if err != nil {
		t.Error("Expected ti-exporter-image to exist:", err)
	}
	if image != "quay.io/auto-lab/jumpstarter-exporter-bootc:0.6.1" {
		t.Errorf("Unexpected ti-exporter-image value: %s", image)
	}

	// Test vault encrypted password
	if !vars.IsVaultEncrypted("snmp_password") {
		t.Error("Expected snmp_password to be vault encrypted")
	}

	if !vars.IsVaultEncrypted("snmp_password") {
		t.Error("Expected snmp_password to be detected as vault encrypted")
	}

	_, err = vars.Get("snmp_password")
	if err == nil {
		t.Error("Expected snmp_password to fail because password is not provided")
	}

}
