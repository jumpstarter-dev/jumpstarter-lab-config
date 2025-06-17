package ssh

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type HostManager interface {
	Status() (string, error)
	NeedsUpdate() (bool, error)
	Diff() (string, error)
	Apply() error
}

type SSHHostManager struct {
	ExporterHost *v1alpha1.ExporterHost `json:"exporterHost,omitempty"`
	sshClient    *ssh.Client
	sftpClient   *sftp.Client
	mutex        *sync.Mutex
}

func NewSSHHostManager(exporterHost *v1alpha1.ExporterHost) (HostManager, error) {

	sshHm := &SSHHostManager{
		ExporterHost: exporterHost,
		mutex:        &sync.Mutex{},
		sshClient:    nil,
		sftpClient:   nil,
	}

	sshClient, err := sshHm.createSshClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client for %q: %w", exporterHost.Name, err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close() // Close SSH client if SFTP client creation fails
		return nil, fmt.Errorf("failed to create SFTP client for %q: %w", exporterHost.Name, err)
	}

	sshHm.sshClient = sshClient
	sshHm.sftpClient = sftpClient
	return sshHm, nil
}

func (m *SSHHostManager) Status() (string, error) {
	if m.ExporterHost.Spec.Management.SSH.Host == "" {
		return "", fmt.Errorf("SSH host is not configured for exporter %s", m.ExporterHost.Name)
	}
	return "Not implemented yet", nil
}

func (m *SSHHostManager) NeedsUpdate() (bool, error) {
	return false, nil
}

func (m *SSHHostManager) Diff() (string, error) {
	return "Not implemented yet", nil
}

func (m *SSHHostManager) Apply() error {

	return nil
}

func (m *SSHHostManager) createSshClient() (*ssh.Client, error) {

	port := 22

	if m.ExporterHost.Spec.Management.SSH.Port != 0 {
		port = m.ExporterHost.Spec.Management.SSH.Port
	}

	// Create SSH client authentication methods
	auth := []ssh.AuthMethod{}
	if m.ExporterHost.Spec.Management.SSH.KeyFile != "" {
		key, err := os.ReadFile(m.ExporterHost.Spec.Management.SSH.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if m.ExporterHost.Spec.Management.SSH.Password != "" {
		auth = append(auth, ssh.Password(m.ExporterHost.Spec.Management.SSH.Password))
	}

	// Check if SSH agent is running and use it if available
	agentSocket := os.Getenv("SSH_AUTH_SOCK")
	if agentSocket != "" {
		log.Fatal("SSH_AUTH_SOCK environment variable not set. Is ssh-agent running?")

		// Connect to the agent's socket.
		conn, err := net.Dial("unix", agentSocket)
		if err != nil {
			log.Fatalf("Failed to open SSH_AUTH_SOCK: %v", err)
		}
		defer conn.Close() // nolint:errcheck

		// Create a new agent client.
		agentClient := agent.NewClient(conn)

		auth = append(auth, ssh.PublicKeysCallback(agentClient.Signers))
	}

	config := &ssh.ClientConfig{
		User:            m.ExporterHost.Spec.Management.SSH.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Use a secure callback in production
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", m.ExporterHost.Spec.Management.SSH.Host, port), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH host %s:%d: %w", m.ExporterHost.Spec.Management.SSH.Host, port, err)
	}
	return client, nil

}
