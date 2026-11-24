package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// Serve SFTP in memory and acknowledge shell requests without executing them.
func rotationTestHost(t *testing.T, failCommand string) (*SSHHostManager, <-chan string) {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(key)
	require.NoError(t, err)
	serverConfig := &gossh.ServerConfig{NoClientAuth: true}
	serverConfig.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	commands := make(chan string, 8)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		server, channels, requests, err := gossh.NewServerConn(conn, serverConfig)
		if err != nil {
			_ = conn.Close()
			return
		}
		defer func() { _ = server.Close() }()
		go gossh.DiscardRequests(requests)
		for incoming := range channels {
			channel, requests, err := incoming.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = channel.Close() }()
				for request := range requests {
					switch request.Type {
					case "subsystem":
						_ = request.Reply(true, nil)
						s := sftp.NewRequestServer(channel, sftp.InMemHandler())
						_ = s.Serve()
						_ = s.Close()
						return
					case "exec":
						var payload struct{ Command string }
						_ = gossh.Unmarshal(request.Payload, &payload)
						commands <- payload.Command
						_ = request.Reply(true, nil)
						var status uint32
						if failCommand != "" && strings.Contains(payload.Command, failCommand) {
							status = 1
						}
						_, _ = channel.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{status}))
						return
					}
				}
			}()
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", "")
	host := createTestExporterHost("sidekick")
	host.Spec.Management.SSH.Host = "127.0.0.1"
	host.Spec.Management.SSH.Port = listener.Addr().(*net.TCPAddr).Port
	m, err := NewSSHHostManager(host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })
	require.NoError(t, m.sftpClient.MkdirAll("/etc/jumpstarter/exporters"))
	return m, commands
}

func TestRotateExporterCredentials(t *testing.T) {
	const filename = "/etc/jumpstarter/exporters/board.yaml"
	const config = "# keep this comment\ntoken: old-token\nendpoint: controller:443\ndriver:\n  cls: example.Driver\n"
	for _, scenario := range []string{"rotate", "dry run", "preflight fails", "missing config", "invalid config", "rotation fails", "restart fails"} {
		t.Run(scenario, func(t *testing.T) {
			failCommand := ""
			switch scenario {
			case "preflight fails":
				failCommand = "test -w"
			case "restart fails":
				failCommand = "SIGHUP"
			}
			m, commands := rotationTestHost(t, failCommand)
			content := config
			if scenario == "invalid config" {
				content = "token: [invalid-token]\n"
			}
			if scenario != "missing config" {
				f, err := m.sftpClient.Create(filename)
				require.NoError(t, err)
				_, err = f.Write([]byte(content))
				require.NoError(t, err)
				require.NoError(t, f.Chmod(0600))
				require.NoError(t, f.Close())
			}
			rotations := 0
			err := m.RotateCredentials("board", scenario == "dry run", func() (string, error) {
				rotations++
				// The preflight command must have completed before controller mutation.
				select {
				case command := <-commands:
					require.Contains(t, command, "test -w")
				default:
					t.Fatal("rotation happened before SSH preflight")
				}
				if scenario == "rotation fails" {
					return "", errors.New("controller unavailable")
				}
				return "new-token", nil
			})
			if scenario == "rotate" || scenario == "dry run" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if scenario == "rotate" || scenario == "rotation fails" || scenario == "restart fails" {
				require.Equal(t, 1, rotations)
			} else {
				require.Zero(t, rotations)
			}
			if scenario == "missing config" {
				return
			}
			f, err := m.sftpClient.Open(filename)
			require.NoError(t, err)
			got, err := io.ReadAll(f)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			if scenario == "rotate" || scenario == "restart fails" {
				var before, after map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(config), &before))
				require.NoError(t, yaml.Unmarshal(got, &after))
				before["token"] = "new-token"
				require.Equal(t, before, after)
				require.Contains(t, string(got), "# keep this comment")
				require.Contains(t, <-commands, "SIGHUP")
			} else {
				require.Equal(t, content, string(got))
			}
			files, err := m.sftpClient.ReadDir("/etc/jumpstarter/exporters")
			require.NoError(t, err)
			require.Len(t, files, 1, "staged files must be cleaned up")
		})
	}
}

func TestExporterTokenRejectsAmbiguousConfigs(t *testing.T) {
	for _, content := range []string{"", "[]", "token: old\ntoken: other", "token: old\n---\ntoken: other", "token: [secret]", "driver: {}"} {
		_, _, err := exporterToken([]byte(content))
		require.Error(t, err)
		require.False(t, strings.Contains(err.Error(), "secret"), "errors must not echo config values")
	}
}
