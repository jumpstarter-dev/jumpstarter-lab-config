package ssh

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type HostManager interface {
	Status() (string, error)
	NeedsUpdate() (bool, error)
	Diff() (string, error)
	Apply(exporterConfig *v1alpha1.ExporterConfigTemplate, dryRun bool) error
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
	err = nil

	if exitCode != 0 {
		err = fmt.Errorf("failed to run command for %q: %q", m.ExporterHost.Name, string(stderrBytes))
	}

	return &CommandResult{
		Stdout:   string(stdoutBytes),
		Stderr:   string(stderrBytes),
		ExitCode: exitCode,
	}, err
}

func (m *SSHHostManager) NeedsUpdate() (bool, error) {
	return false, nil
}

func (m *SSHHostManager) Diff() (string, error) {
	return "Not implemented yet", nil
}

func (m *SSHHostManager) Apply(exporterConfig *v1alpha1.ExporterConfigTemplate, dryRun bool) error {
	// Validate mutual exclusivity: both templates cannot be specified simultaneously
	if exporterConfig.Spec.SystemdContainerTemplate != "" && exporterConfig.Spec.SystemdServiceTemplate != "" {
		return fmt.Errorf("both SystemdContainerTemplate and SystemdServiceTemplate specified - only one should be used")
	}

	svcName := exporterConfig.Spec.ExporterMetadata.Name
	containerSystemdFile := "/etc/containers/systemd/" + svcName + ".container"
	serviceSystemdFile := "/etc/systemd/system/" + svcName + ".service"
	exporterConfigFile := "/etc/jumpstarter/exporters/" + svcName + ".yaml"

	changedContainer, err := m.reconcileFile(containerSystemdFile, exporterConfig.Spec.SystemdContainerTemplate, dryRun)
	if err != nil {
		return fmt.Errorf("failed to reconcile container systemd file: %w", err)
	}

	changedService, err := m.reconcileFile(serviceSystemdFile, exporterConfig.Spec.SystemdServiceTemplate, dryRun)
	if err != nil {
		return fmt.Errorf("failed to reconcile service systemd file: %w", err)
	}

	changedExporterConfig, err := m.reconcileFile(exporterConfigFile, exporterConfig.Spec.ConfigTemplate, dryRun)
	if err != nil {
		return fmt.Errorf("failed to reconcile exporter config file: %w", err)
	}

	// if any of the files changed, reload systemd, enable service and restart the exporter
	if (changedExporterConfig || changedContainer || changedService) && !dryRun {
		_, err := m.runCommand("systemctl daemon-reload")
		if err != nil {
			return fmt.Errorf("failed to reload systemd: %w", err)
		}
		_, err = m.runCommand("systemctl start " + svcName)
		if err != nil {
			return fmt.Errorf("failed to enable exporter: %w", err)
		}

		_, err = m.runCommand("systemctl restart " + svcName)
		if err != nil {
			return fmt.Errorf("failed to restart exporter: %w", err)
		}
	}

	return nil
}

// sanitizeDiff removes sensitive information from diff output
func sanitizeDiff(diff string) string {
	// Regex patterns to match sensitive fields
	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Match patterns like "token: value", "password: value", etc.
		{`(?i)(token|password|key|secret|api_key|auth_token|bearer_token|access_token|refresh_token|client_secret|private_key|passphrase|credential)(\s*[:=]\s*)([^\s\n]+)`, `${1}${2}<TOKEN>`},
		// Match patterns like "TOKEN=value", "PASSWORD=value", etc.
		{`(?i)(TOKEN|PASSWORD|KEY|SECRET|API_KEY|AUTH_TOKEN|BEARER_TOKEN|ACCESS_TOKEN|REFRESH_TOKEN|CLIENT_SECRET|PRIVATE_KEY|PASSPHRASE|CREDENTIAL)(\s*=\s*)([^\s\n]+)`, `${1}${2}<TOKEN>`},
		// Match patterns in double quotes like "token": "value"
		{`(?i)(")(token|password|key|secret|api_key|auth_token|bearer_token|access_token|refresh_token|client_secret|private_key|passphrase|credential)("\s*[:=]\s*")([^"]+)(")`, `${1}${2}${3}<TOKEN>${5}`},
		// Match patterns in single quotes like 'token': 'value'
		{`(?i)(')(token|password|key|secret|api_key|auth_token|bearer_token|access_token|refresh_token|client_secret|private_key|passphrase|credential)('\s*[:=]\s*')([^']+)(')`, `${1}${2}${3}<TOKEN>${5}`},
	}

	result := diff
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		result = re.ReplaceAllString(result, p.replacement)
	}

	return result
}

func (m *SSHHostManager) reconcileFile(path string, content string, dryRun bool) (bool, error) {
	// Check if file exists and read its content
	file, err := m.sftpClient.Open(path)
	if err != nil {
		// File doesn't exist
		if content == "" {
			// File doesn't exist and content is empty - nothing to do
			return false, nil
		}

		// File doesn't exist and content is not empty - create it
		if dryRun {
			fmt.Printf("            📄 Would create file: %s\n", path)
			return true, nil
		}

		// Create parent directories if needed
		parentDir := filepath.Dir(path)
		if parentDir != "/" && parentDir != "." {
			err = m.sftpClient.MkdirAll(parentDir)
			if err != nil {
				return false, fmt.Errorf("failed to create parent directories for %s: %w", path, err)
			}
		}

		// Create the file
		newFile, err := m.sftpClient.Create(path)
		if err != nil {
			return false, fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer func() {
			_ = newFile.Close() // nolint:errcheck
		}()

		_, err = newFile.Write([]byte(content))
		if err != nil {
			return false, fmt.Errorf("failed to write content to %s: %w", path, err)
		}

		fmt.Printf("            📄 Created file: %s\n", path)
		return true, nil
	}

	// File exists, read its content
	existingContent, err := io.ReadAll(file)
	_ = file.Close() // nolint:errcheck
	if err != nil {
		fmt.Printf("Failed to read existing file %s: %v\n", path, err)
		return false, fmt.Errorf("failed to read existing file %s: %w", path, err)
	}

	// If content is empty, delete the file
	if content == "" {
		if dryRun {
			fmt.Printf("            🗑️ Would delete file: %s\n", path)
			return true, nil
		}
		err = m.sftpClient.Remove(path)
		if err != nil {
			return false, fmt.Errorf("failed to delete file %s: %w", path, err)
		}
		fmt.Printf("            🗑️  Deleted file: %s\n", path)
		return true, nil
	}

	// Check if content matches
	if string(existingContent) == content {
		// Content matches, no change needed
		return false, nil
	}

	// Content doesn't match, show diff and update the file
	diff := cmp.Diff(string(existingContent), content)
	if diff != "" {
		sanitizedDiff := sanitizeDiff(diff)
		fmt.Printf("            📄 Changes for file: %s\n", path)
		fmt.Printf("            Diff (-existing +new):\n%s\n", sanitizedDiff)
	}

	if dryRun {
		fmt.Printf("            ✏️ Would update file: %s\n", path)
		return true, nil
	}

	updateFile, err := m.sftpClient.OpenFile(path, os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return false, fmt.Errorf("failed to open file %s for writing: %w", path, err)
	}
	defer func() {
		_ = updateFile.Close() // nolint:errcheck
	}()

	_, err = updateFile.Write([]byte(content))
	if err != nil {
		fmt.Printf("Failed to write updated content to %s: %v\n", path, err)
		return false, fmt.Errorf("failed to write updated content to %s: %w", path, err)
	}

	fmt.Printf("            ✏️ Updated file: %s\n", path)
	return true, nil
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
