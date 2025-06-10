package vars

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Variables represents a collection of variables loaded from a YAML file
type Variables struct {
	decryptor *VaultDecryptor // Optional decryptor for vault-encrypted variables
	data      map[string]interface{}
}

func NewVariables(vaultPasswordFile string) (*Variables, error) {
	var decryptor *VaultDecryptor

	if vaultPasswordFile == "" {
		if os.Getenv("ANSIBLE_VAULT_PASSWORD_FILE") != "" {
			vaultPasswordFile = os.Getenv("ANSIBLE_VAULT_PASSWORD_FILE")
		} else if os.Getenv("ANSIBLE_VAULT_PASSWORD") != "" {
			// If ANSIBLE_VAULT_PASSWORD is set, use it directly
			decryptor = NewVaultDecryptor(os.Getenv("ANSIBLE_VAULT_PASSWORD"))
		}
	}

	if vaultPasswordFile != "" {
		password, err := os.ReadFile(vaultPasswordFile)
		if err != nil {
			return nil, fmt.Errorf("error reading vault password file %s: %w", vaultPasswordFile, err)
		}
		decryptor = NewVaultDecryptor(string(password))
	}

	return &Variables{
		decryptor: decryptor,
		data:      make(map[string]interface{}),
	}, nil
}

// LoadFromFile loads variables from a YAML file
func (v *Variables) LoadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading variables file %s: %w", filePath, err)
	}

	var varData map[string]interface{}
	err = yaml.Unmarshal(data, &varData)
	if err != nil {
		return fmt.Errorf("error parsing YAML from file %s: %w", filePath, err)
	}

	// Merge the loaded variables into the existing data, failing if a key already exists
	for key, value := range varData {
		if _, exists := v.data[key]; exists {
			return fmt.Errorf("variable %s already exists, cannot overwrite with variable from file: %s", key, filePath)
		}
		v.data[key] = value
	}
	return nil
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

// GetDecrypted retrieves and decrypts a vault-encrypted variable
func (v *Variables) Get(key string) (string, error) {
	value, exists := v.data[key]
	if !exists {
		return "", fmt.Errorf("variable %s not found", key)
	}
	strValue, ok := value.(string)

	if !ok {
		return "", fmt.Errorf("variable %s is not a string", key)
	}

	if !v.IsVaultEncrypted(key) {
		return strValue, nil
	}

	if v.decryptor == nil {
		return "", fmt.Errorf("ANSIBLE_VAULT_PASSWORD_FILE or ANSIBLE_VAULT_PASSWORD required for encrypted key %s", key)
	}

	return v.decryptor.Decrypt(strValue)
}

// mostly used for testing purposes
func (v *Variables) Set(key string, value string) error {
	if _, exists := v.data[key]; exists {
		return fmt.Errorf("variable %s already exists, cannot overwrite", key)
	}
	v.data[key] = value
	return nil
}
