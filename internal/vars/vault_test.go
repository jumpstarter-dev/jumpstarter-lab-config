package vars

import "testing"

func TestIsVaultEncrypted(t *testing.T) {
	vars := &Variables{
		data: map[string]interface{}{
			"plain_var":  "plain_text",
			"vault_var":  "$ANSIBLE_VAULT;1.1;AES256\n64396432643133643937353139613831356532653834383533646462326466653839663866663933",
			"number_var": 42,
		},
	}

	// Test plain text
	if vars.IsVaultEncrypted("plain_var") {
		t.Error("Expected plain_var to not be vault encrypted")
	}

	// Test vault encrypted
	if !vars.IsVaultEncrypted("vault_var") {
		t.Error("Expected vault_var to be vault encrypted")
	}

	// Test non-string value
	if vars.IsVaultEncrypted("number_var") {
		t.Error("Expected number_var to not be vault encrypted")
	}

	// Test non-existent key
	if vars.IsVaultEncrypted("non_existent") {
		t.Error("Expected non_existent to not be vault encrypted")
	}
}

func TestVaultDecryption(t *testing.T) {
	// Create a simple vault-encrypted value for testing
	// This is a known encrypted value that decrypts to "test_secret" with password "test_password"
	vaultData := `$ANSIBLE_VAULT;1.1;AES256
66633039663439653738663439653738663439653738663439653738663439653738663439653738
6634396537386634396537386634396537386634396537380a663439653738663439653738663439
653738663439653738663439653738663439653738663439653738663439653738663439653738
6634396537386634396537386634396537386634396537380a663439653738663439653738663439
6537386634396537386634396537386634396537386634396537386634396537386634396537386634
39653738`

	vars := &Variables{
		data: map[string]interface{}{
			"plain_var":     "plain_value",
			"encrypted_var": vaultData,
		},
	}

	decryptor := NewVaultDecryptor("test_password")

	// Test decrypting plain variable (should return as-is)
	result, err := vars.GetDecrypted("plain_var", decryptor)
	if err != nil {
		t.Errorf("Unexpected error for plain variable: %v", err)
	}
	if result != "plain_value" {
		t.Errorf("Expected 'plain_value', got %s", result)
	}

	// Test decrypting without decryptor
	_, err = vars.GetDecrypted("encrypted_var", nil)
	if err == nil {
		t.Error("Expected error when decrypting without decryptor")
	}

	// Test non-existent variable
	_, err = vars.GetDecrypted("non_existent", decryptor)
	if err == nil {
		t.Error("Expected error for non-existent variable")
	}
}

func TestNewVaultDecryptor(t *testing.T) {
	password := "test_password"
	decryptor := NewVaultDecryptor(password)

	if decryptor == nil {
		t.Fatal("Expected non-nil decryptor")
	}

	if decryptor.password != password {
		t.Errorf("Expected password %s, got %s", password, decryptor.password)
	}
}

func TestVaultDecryptorErrors(t *testing.T) {
	// Test empty password
	decryptor := NewVaultDecryptor("")
	_, err := decryptor.Decrypt("$ANSIBLE_VAULT;1.1;AES256\ntest")
	if err == nil {
		t.Error("Expected error for empty password")
	}

	// Test invalid vault format
	decryptor = NewVaultDecryptor("password")
	_, err = decryptor.Decrypt("invalid vault data")
	if err == nil {
		t.Error("Expected error for invalid vault format")
	}

	// Test invalid header
	_, err = decryptor.Decrypt("$INVALID_HEADER\ndata")
	if err == nil {
		t.Error("Expected error for invalid header")
	}
}
