package vars

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Variables represents a collection of variables loaded from a YAML file
type Variables struct {
	data map[string]interface{}
}

// LoadFromFile loads variables from a YAML file
func LoadFromFile(filePath string) (*Variables, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading variables file %s: %w", filePath, err)
	}

	var varData map[string]interface{}
	err = yaml.Unmarshal(data, &varData)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML from file %s: %w", filePath, err)
	}

	return &Variables{
		data: varData,
	}, nil
}

// Get retrieves a variable value by key
func (v *Variables) Get(key string) (interface{}, bool) {
	value, exists := v.data[key]
	return value, exists
}

// GetString retrieves a variable value as a string
func (v *Variables) GetString(key string) (string, bool) {
	value, exists := v.data[key]
	if !exists {
		return "", false
	}
	
	if str, ok := value.(string); ok {
		return str, true
	}
	
	return "", false
}

// IsVaultEncrypted checks if a variable value is Ansible Vault encrypted
func (v *Variables) IsVaultEncrypted(key string) bool {
	value, exists := v.GetString(key)
	if !exists {
		return false
	}
	
	return strings.HasPrefix(value, "$ANSIBLE_VAULT;")
}

// GetAllKeys returns all variable keys
func (v *Variables) GetAllKeys() []string {
	keys := make([]string, 0, len(v.data))
	for key := range v.data {
		keys = append(keys, key)
	}
	return keys
}

// Has checks if a variable key exists
func (v *Variables) Has(key string) bool {
	_, exists := v.data[key]
	return exists
}
