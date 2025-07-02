package ssh

import (
	"fmt"
	"io"
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

// CommandResult represents the result of running a command via SSH
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
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
	result, err := m.runCommand("ls -la")
	if err != nil {
		return "", fmt.Errorf("failed to run status command for %q: %w", m.ExporterHost.Name, err)
	}

	// For now, return a simple status based on exit code
	if result.ExitCode == 0 {
		return "ok", nil
	}
	return fmt.Sprintf("error (exit code: %d)", result.ExitCode), nil
}

// runCommand executes a command on the remote host and returns the result
func (m *SSHHostManager) runCommand(command string) (*CommandResult, error) {
	if m.sshClient == nil {
		return nil, fmt.Errorf("sshClient is not initialized")
	}
	session, err := m.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session for %q: %w", m.ExporterHost.Name, err)
	}
	defer func() {
		_ = session.Close() // nolint:errcheck
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe for %q: %w", m.ExporterHost.Name, err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe for %q: %w", m.ExporterHost.Name, err)
	}

	// Capture stdout and stderr
	var stdoutBytes, stderrBytes []byte
	var stdoutErr, stderrErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutBytes, stdoutErr = io.ReadAll(stdout)
	}()
	go func() {
		defer wg.Done()
		stderrBytes, stderrErr = io.ReadAll(stderr)
	}()

	// Run the command
	err = session.Run(command)

	// Wait for stdout and stderr to be read
	wg.Wait()

	// Check for errors in reading stdout/stderr
	if stdoutErr != nil {
		return nil, fmt.Errorf("failed to read stdout for %q: %w", m.ExporterHost.Name, stdoutErr)
	}
	if stderrErr != nil {
		return nil, fmt.Errorf("failed to read stderr for %q: %w", m.ExporterHost.Name, stderrErr)
	}

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			// If it's not an exit error, return the error
			return nil, fmt.Errorf("failed to run command for %q: %w", m.ExporterHost.Name, err)
		}
	}

	return &CommandResult{
		Stdout:   string(stdoutBytes),
		Stderr:   string(stderrBytes),
		ExitCode: exitCode,
	}, nil
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
		var signer ssh.Signer
		if m.ExporterHost.Spec.Management.SSH.SSHKeyPassword != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(m.ExporterHost.Spec.Management.SSH.SSHKeyPassword))
			if err != nil {
				return nil, fmt.Errorf("failed to parse encrypted SSH private key from file: %w", err)
			}
		} else {
			signer, err = ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SSH private key from file: %w", err)
			}
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if m.ExporterHost.Spec.Management.SSH.SSHKeyData != "" {
		var signer ssh.Signer
		var err error
		if m.ExporterHost.Spec.Management.SSH.SSHKeyPassword != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(m.ExporterHost.Spec.Management.SSH.SSHKeyData), []byte(m.ExporterHost.Spec.Management.SSH.SSHKeyPassword))
			if err != nil {
				return nil, fmt.Errorf("failed to parse encrypted SSH private key from sshKeyData: %w", err)
			}
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(m.ExporterHost.Spec.Management.SSH.SSHKeyData))
			if err != nil {
				return nil, fmt.Errorf("failed to parse SSH private key from sshKeyData: %w", err)
			}
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if m.ExporterHost.Spec.Management.SSH.Password != "" {
		auth = append(auth, ssh.Password(m.ExporterHost.Spec.Management.SSH.Password))
	}

	// Check if SSH agent is running and use it if available
	agentSocket := os.Getenv("SSH_AUTH_SOCK")
	if agentSocket != "" {
		// Connect to the agent's socket.
		conn, err := net.Dial("unix", agentSocket)
		if err != nil {
			log.Printf("Failed to connect to SSH agent: %v", err)
		} else {
			defer conn.Close() // nolint:errcheck

			// Create a new agent client.
			agentClient := agent.NewClient(conn)

			auth = append(auth, ssh.PublicKeysCallback(agentClient.Signers))
		}
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
