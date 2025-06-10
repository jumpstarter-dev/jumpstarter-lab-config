package vars

import "testing"

// test_secret encrypted with password "test_password"
const VAULT_DATA = `$ANSIBLE_VAULT;1.1;AES256
66313930643739626237323731663838613636363336346334356662623932653534306263656233
3230316330383937363134383661353534393139393263620a396462343137316236633438316635
66613264356433323739376564666632353761633965363665363737333463653339386339336361
3232336361363734340a373364626431333937363662353139303637303435373132366434313139
3132`

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

	vars := &Variables{
		data: map[string]interface{}{
			"plain_var":     "plain_value",
			"encrypted_var": VAULT_DATA,
		},
		decryptor: NewVaultDecryptor("test_password"),
	}

	// Test decrypting plain variable (should return as-is)
	result, err := vars.Get("plain_var")
	if err != nil {
		t.Errorf("Unexpected error for plain variable: %v", err)
	}
	if result != "plain_value" {
		t.Errorf("Expected 'plain_value', got %s", result)
	}

	// Test decrypting without decryptor
	decrypted, err := vars.Get("encrypted_var")
	if err != nil {
		t.Error("Expected decryption to work", err)
	}

	if decrypted != "test_secret" {
		t.Errorf("Expected decrypted value to be 'test_secret', got %s", decrypted)
	}

	// Test non-existent variable
	_, err = vars.Get("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent variable")
	}
}

func TestVaultDecryptionPasswdFile(t *testing.T) {
	// Create a simple vault-encrypted value for testing
	// This is a known encrypted value that decrypts to "test_secret" with password "test_password"

	vars := &Variables{
		data: map[string]interface{}{
			"plain_var":     "plain_value",
			"encrypted_var": VAULT_DATA,
		},
		decryptor: NewVaultDecryptor("test_password"),
	}

	// Test decrypting plain variable (should return as-is)
	result, err := vars.Get("plain_var")
	if err != nil {
		t.Errorf("Unexpected error for plain variable: %v", err)
	}
	if result != "plain_value" {
		t.Errorf("Expected 'plain_value', got %s", result)
	}

	// Test decrypting without decryptor
	_, err = vars.Get("encrypted_var")
	if err != nil {
		t.Error("Expected decryption to work: ", err)
	}

	// Test non-existent variable
	_, err = vars.Get("non_existent")
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
