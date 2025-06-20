package vars

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// NOTE, vault decryption is based on the code from https://github.com/liangrog/vault
// also under the Apache License 2.0 but written by @lianrog, I am not importing
// his package to avoid dependency issues on security libraries, where the maintenance
// of the project seems to be low (years since last commit).

const (
	// Ansible Vault constants
	vaultHeader     = "$ANSIBLE_VAULT;1.1;AES256"
	cipherKeyLength = 32
	hmacKeyLength   = 32
	ivLength        = 16
	saltLength      = 32
	iteration       = 10000
	charPerLine     = 80
)

// CipherKey holds the keys needed for vault decryption
type CipherKey struct {
	Key     []byte
	HMACKey []byte
	IV      []byte
}

// VaultDecryptor handles Ansible Vault decryption
type VaultDecryptor struct {
	password string
}

// NewVaultDecryptor creates a new vault decryptor with the given password
func NewVaultDecryptor(password string) *VaultDecryptor {
	// trim the password left and right of whitespace, line breaks and tabs
	password = strings.TrimSpace(password)
	password = strings.ReplaceAll(password, "\n", "")
	password = strings.ReplaceAll(password, "\r", "")
	password = strings.ReplaceAll(password, "\t", "")
	return &VaultDecryptor{password: password}
}

// IsVaultEncrypted checks if a variable value is Ansible Vault encrypted
func (v *Variables) IsVaultEncrypted(key string) bool {
	value, exists := v.data[key]
	if !exists {
		return false
	}
	// Check if the value is a string and starts with the vault header
	if strValue, ok := value.(string); ok {
		return strings.HasPrefix(strValue, "$ANSIBLE_VAULT;")
	} else {
		return false // Not a string, cannot be vault encrypted
	}
}

// Decrypt decrypts an Ansible Vault encrypted string
func (vd *VaultDecryptor) Decrypt(vaultData string) (string, error) {
	if vd.password == "" {
		return "", errors.New("empty password")
	}

	// Parse vault data
	salt, checkSum, encryptedData, err := parseVaultData(vaultData)
	if err != nil {
		return "", err
	}
	// Generate keys
	key := generateCipherKey(vd.password, salt)

	// Verify checksum
	if !isChecksumValid(checkSum, encryptedData, key.HMACKey) {
		return "", errors.New("checksum doesn't match")
	}

	// Decrypt data
	decrypted, err := decryptData(encryptedData, key)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// parseVaultData parses the vault file format and extracts components
func parseVaultData(vaultData string) (salt, checkSum, data []byte, err error) {
	lines := strings.SplitN(vaultData, "\n", 2)
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != vaultHeader {
		return nil, nil, nil, errors.New("invalid vault file format")
	}

	// Concat all content lines and remove whitespace
	content := strings.TrimSpace(lines[1])
	content = strings.ReplaceAll(content, "\r", "")
	content = strings.ReplaceAll(content, "\n", "")

	// Decode the hex content
	decodedBytes, err := hex.DecodeString(content)
	if err != nil {
		return nil, nil, nil, err
	}

	// Split into components
	components := strings.Split(string(decodedBytes), "\n")
	if len(components) != 3 {
		return nil, nil, nil, errors.New("invalid encoded data")
	}

	salt, err = hex.DecodeString(components[0])
	if err != nil {
		return nil, nil, nil, err
	}

	checkSum, err = hex.DecodeString(components[1])
	if err != nil {
		return nil, nil, nil, err
	}

	data, err = hex.DecodeString(components[2])
	if err != nil {
		return nil, nil, nil, err
	}

	return salt, checkSum, data, nil
}

// generateCipherKey generates cipher keys from password and salt
func generateCipherKey(password string, salt []byte) *CipherKey {
	k := pbkdf2.Key(
		[]byte(password),
		salt,
		iteration,
		(cipherKeyLength + hmacKeyLength + ivLength),
		sha256.New,
	)

	return &CipherKey{
		Key:     k[:cipherKeyLength],
		HMACKey: k[cipherKeyLength:(cipherKeyLength + hmacKeyLength)],
		IV:      k[(cipherKeyLength + hmacKeyLength):(cipherKeyLength + hmacKeyLength + ivLength)],
	}
}

// isChecksumValid validates HMAC checksum
func isChecksumValid(checkSum, data, hmacKey []byte) bool {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(data)
	return hmac.Equal(mac.Sum(nil), checkSum)
}

// decryptData decrypts the encrypted data using AES-CTR
func decryptData(data []byte, key *CipherKey) ([]byte, error) {
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, key.IV)
	decryptedData := make([]byte, len(data))
	stream.XORKeyStream(decryptedData, data)

	// Unpad the decrypted data
	return aesBlockUnpad(decryptedData)
}

// aesBlockUnpad removes AES block padding
func aesBlockUnpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("empty data")
	}

	unpad := int(data[length-1])
	if unpad > length {
		return nil, errors.New("unpad error")
	}

	return data[:(length - unpad)], nil
}
