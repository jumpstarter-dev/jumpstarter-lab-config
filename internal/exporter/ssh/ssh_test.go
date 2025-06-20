package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testPassword = "testpass"

// Test data helpers
func generateTestSSHKey(t *testing.T, passphrase string) []byte {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Convert to PEM format
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	var privateKeyBlock *pem.Block

	if passphrase != "" {
		//nolint:staticcheck // SA1019: Using deprecated EncryptPEMBlock for testing encrypted keys
		privateKeyBlock, err = x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", privateKeyDER, []byte(passphrase), x509.PEMCipherAES256)
		if err != nil {
			t.Fatalf("Failed to encrypt private key: %v", err)
		}
	} else {
		privateKeyBlock = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyDER,
		}
	}

	return pem.EncodeToMemory(privateKeyBlock)
}

func createTestExporterHost(name string) *v1alpha1.ExporterHost {
	return &v1alpha1.ExporterHost{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ExporterHostSpec{
			Management: v1alpha1.Management{
				SSH: v1alpha1.SSHCredentials{
					Host: "test-host.example.com",
					User: "testuser",
					Port: 22,
				},
			},
		},
	}
}

func TestSSHHostManagerCreation(t *testing.T) {
	tests := []struct {
		name        string
		setupHost   func() *v1alpha1.ExporterHost
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid host configuration",
			setupHost: func() *v1alpha1.ExporterHost {
				host := createTestExporterHost("test-host")
				host.Spec.Management.SSH.Password = testPassword
				return host
			},
			expectError: true, // Will fail because we can't actually connect
			errorMsg:    "failed to connect to SSH host",
		},
		{
			name: "missing SSH host",
			setupHost: func() *v1alpha1.ExporterHost {
				host := createTestExporterHost("test-host")
				host.Spec.Management.SSH.Host = ""
				return host
			},
			expectError: true,
			errorMsg:    "failed to create SSH client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := tt.setupHost()
			_, err := NewSSHHostManager(host)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAuthMethodCreation(t *testing.T) {
	tests := []struct {
		name        string
		setupHost   func(t *testing.T) *v1alpha1.ExporterHost
		expectError bool
		description string
	}{
		{
			name: "password authentication",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				host := createTestExporterHost("password-test")
				host.Spec.Management.SSH.Password = "testpassword"
				return host
			},
			expectError: false,
			description: "Should create password auth method",
		},
		{
			name: "key file authentication",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				// Create temporary key file
				tempDir := t.TempDir()
				keyFile := filepath.Join(tempDir, "test_key")

				privateKeyPEM := generateTestSSHKey(t, "")
				err := os.WriteFile(keyFile, privateKeyPEM, 0600)
				if err != nil {
					t.Fatalf("Failed to write key file: %v", err)
				}

				host := createTestExporterHost("keyfile-test")
				host.Spec.Management.SSH.KeyFile = keyFile
				return host
			},
			expectError: false,
			description: "Should create public key auth from file",
		},
		{
			name: "encrypted key file authentication",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				// Create temporary encrypted key file
				tempDir := t.TempDir()
				keyFile := filepath.Join(tempDir, "test_key_encrypted")

				privateKeyPEM := generateTestSSHKey(t, testPassword)
				err := os.WriteFile(keyFile, privateKeyPEM, 0600)
				if err != nil {
					t.Fatalf("Failed to write encrypted key file: %v", err)
				}

				host := createTestExporterHost("encrypted-keyfile-test")
				host.Spec.Management.SSH.KeyFile = keyFile
				host.Spec.Management.SSH.SSHKeyPassword = testPassword
				return host
			},
			expectError: false,
			description: "Should create public key auth from encrypted file",
		},
		{
			name: "sshKeyData authentication",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				privateKeyPEM := generateTestSSHKey(t, "")

				host := createTestExporterHost("keydata-test")
				host.Spec.Management.SSH.SSHKeyData = string(privateKeyPEM)
				return host
			},
			expectError: false,
			description: "Should create public key auth from key data",
		},
		{
			name: "encrypted sshKeyData authentication",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				privateKeyPEM := generateTestSSHKey(t, testPassword)

				host := createTestExporterHost("encrypted-keydata-test")
				host.Spec.Management.SSH.SSHKeyData = string(privateKeyPEM)
				host.Spec.Management.SSH.SSHKeyPassword = testPassword
				return host
			},
			expectError: false,
			description: "Should create public key auth from encrypted key data",
		},
		{
			name: "invalid key file path",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				host := createTestExporterHost("invalid-keyfile-test")
				host.Spec.Management.SSH.KeyFile = "/nonexistent/key/file"
				return host
			},
			expectError: true,
			description: "Should fail with invalid key file path",
		},
		{
			name: "invalid key data",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				host := createTestExporterHost("invalid-keydata-test")
				host.Spec.Management.SSH.SSHKeyData = "invalid-key-data"
				return host
			},
			expectError: true,
			description: "Should fail with invalid key data",
		},
		{
			name: "wrong password for encrypted key",
			setupHost: func(t *testing.T) *v1alpha1.ExporterHost {
				privateKeyPEM := generateTestSSHKey(t, "correctpass")

				host := createTestExporterHost("wrong-password-test")
				host.Spec.Management.SSH.SSHKeyData = string(privateKeyPEM)
				host.Spec.Management.SSH.SSHKeyPassword = "wrongpass"
				return host
			},
			expectError: true,
			description: "Should fail with wrong password for encrypted key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := tt.setupHost(t)

			sshManager := &SSHHostManager{
				ExporterHost: host,
			}

			_, err := sshManager.createSshClient()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for %s", tt.description)
				}
			} else {
				// We expect the connection to fail (no real SSH server),
				// but authentication setup should succeed
				if err != nil && !strings.Contains(err.Error(), "failed to connect to SSH host") {
					t.Errorf("Unexpected error type: %v", err)
				}
			}
		})
	}
}

func TestSSHCredentialsValidation(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		user         string
		port         int
		expectError  bool
		expectedHost string
		expectedPort int
	}{
		{
			name:         "default port",
			host:         "test.example.com",
			user:         "testuser",
			port:         0,
			expectError:  false,
			expectedHost: "test.example.com",
			expectedPort: 22,
		},
		{
			name:         "custom port",
			host:         "test.example.com",
			user:         "testuser",
			port:         2222,
			expectError:  false,
			expectedHost: "test.example.com",
			expectedPort: 2222,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := createTestExporterHost("port-test")
			host.Spec.Management.SSH.Host = tt.host
			host.Spec.Management.SSH.User = tt.user
			host.Spec.Management.SSH.Port = tt.port
			host.Spec.Management.SSH.Password = testPassword // Add auth method

			sshManager := &SSHHostManager{
				ExporterHost: host,
			}

			_, err := sshManager.createSshClient()

			// We expect connection to fail, but we can verify the error message contains the expected host:port
			if err != nil {
				expectedAddress := fmt.Sprintf("%s:%d", tt.expectedHost, tt.expectedPort)
				if !strings.Contains(err.Error(), expectedAddress) {
					t.Errorf("Expected error to contain %q, got %q", expectedAddress, err.Error())
				}
			}
		})
	}
}

func TestSSHHostManagerInterface(t *testing.T) {
	host := createTestExporterHost("interface-test")
	host.Spec.Management.SSH.Password = testPassword

	// Test that SSHHostManager implements HostManager interface
	var manager HostManager
	sshManager := &SSHHostManager{
		ExporterHost: host,
	}

	manager = sshManager

	// Test interface methods are available
	_, err := manager.Status()
	if err != nil && !strings.Contains(err.Error(), "SSH host is not configured") && !strings.Contains(err.Error(), "sshClient is not initialized") {
		// We expect either success or the specific SSH host error
		t.Errorf("Unexpected error from Status method: %v", err)
	}
}
