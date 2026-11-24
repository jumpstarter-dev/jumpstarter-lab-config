package ssh

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/validation"
)

// RotateCredentials checks the installed config before invoking rotate, then
// replaces only its token. Keep this SSH connection open throughout rotation.
func (m *SSHHostManager) RotateCredentials(service string, dryRun bool, rotate func() (string, error)) error {
	if len(validation.IsDNS1123Subdomain(service)) != 0 {
		return fmt.Errorf("invalid exporter service name %q", service)
	}
	filename := "/etc/jumpstarter/exporters/" + service + ".yaml"
	info, err := m.sftpClient.Lstat(filename)
	if err != nil {
		return fmt.Errorf("check installed exporter config: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("exporter config %s must be a regular file", filename)
	}
	f, err := m.sftpClient.Open(filename)
	if err != nil {
		return fmt.Errorf("read installed exporter config: %w", err)
	}
	content, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("read installed exporter config: %w", err)
	}
	doc, token, err := exporterToken(content)
	if err != nil {
		return err
	}
	if _, ok := m.sftpClient.HasExtension("posix-rename@openssh.com"); !ok {
		return fmt.Errorf("SFTP server must support atomic POSIX rename")
	}
	// Names are DNS-validated above, so these single-quoted paths are safe.
	_, err = m.runCommand(fmt.Sprintf("test -w '%s' && test -w '%s' && systemctl cat -- '%s' >/dev/null",
		filename, path.Dir(filename), service))
	if err != nil {
		return fmt.Errorf("exporter preflight failed: %w", err)
	}
	if dryRun {
		return nil
	}
	// Stage beside the destination, before rotating, to check SFTP write access.
	staged := filename + ".rotate-" + rand.Text()
	f, err = m.sftpClient.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return fmt.Errorf("stage exporter config: %w", err)
	}
	defer func() { _ = m.sftpClient.Remove(staged) }()
	defer func() { _ = f.Close() }()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if stat, ok := info.Sys().(*sftp.FileStat); ok {
		if err := f.Chown(int(stat.UID), int(stat.GID)); err != nil {
			return fmt.Errorf("preserve exporter config ownership: %w", err)
		}
	}
	newToken, err := rotate()
	if err != nil {
		return err
	}
	if strings.TrimSpace(newToken) == "" {
		return fmt.Errorf("controller returned an empty token")
	}
	token.Value = newToken
	updated, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("could not encode updated exporter config")
	}
	if _, err := f.Write(updated); err != nil {
		return fmt.Errorf("write replacement exporter config: %w", err)
	}
	if err := f.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := m.sftpClient.PosixRename(staged, filename); err != nil {
		return fmt.Errorf("install replacement exporter config: %w", err)
	}
	// Match apply's lease-aware restart, but propagate failures to the operator.
	_, err = m.runCommand(fmt.Sprintf(
		"if systemctl is-active --quiet '%[1]s'; then "+
			"podman kill -s SIGHUP '%[1]s' 2>/dev/null || systemctl kill -s SIGHUP '%[1]s'; "+
			"else systemctl restart '%[1]s'; fi", service))
	if err != nil {
		return fmt.Errorf("token installed, but exporter restart request failed: %w", err)
	}
	return nil
}

func exporterToken(content []byte) (*yaml.Node, *yaml.Node, error) {
	var doc yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	if decoder.Decode(&doc) != nil || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("installed exporter config must be a YAML mapping")
	}
	var extra yaml.Node
	if decoder.Decode(&extra) != io.EOF {
		return nil, nil, fmt.Errorf("installed exporter config must contain one YAML document")
	}
	var token *yaml.Node
	fields := doc.Content[0].Content
	for n := 0; n < len(fields); n += 2 {
		if fields[n].Value == "token" {
			if token != nil {
				return nil, nil, fmt.Errorf("installed exporter config contains duplicate token fields")
			}
			token = fields[n+1]
		}
	}
	if token == nil || token.Kind != yaml.ScalarNode || token.Tag != "!!str" {
		return nil, nil, fmt.Errorf("installed exporter config must have a string token field")
	}
	return &doc, token, nil
}
