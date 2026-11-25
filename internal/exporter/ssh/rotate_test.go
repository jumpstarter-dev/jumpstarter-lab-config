package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
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
						if failCommand == "bootc updating" && strings.Contains(payload.Command, "is-active bootc-") {
							_, _ = io.WriteString(channel, "active\ninactive\n")
						}
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
	const config = "# keep this comment\r\ntoken: \"old-token\"  # credential\r\nendpoint: 'controller:443'\r\ndriver:\r\n  cls: example.Driver\r\n  token: old-token\r\n  data: {hex: 0x10, flag: true}\r\n"
	for _, scenario := range []string{"rotate", "forced restart", "bootc updating", "plain token", "single quoted token", "flow token", "special token", "dry run", "dry run rotation fails", "preflight fails", "missing config", "invalid config", "unsupported token", "rotation fails", "restart fails"} {
		t.Run(scenario, func(t *testing.T) {
			failCommand := ""
			switch scenario {
			case "preflight fails":
				failCommand = "test -w"
			case "restart fails":
				failCommand = "SIGHUP"
			case "bootc updating":
				failCommand = scenario
			}
			m, commands := rotationTestHost(t, failCommand)
			content := config
			success := slices.Contains([]string{"rotate", "forced restart", "plain token", "single quoted token", "flow token", "special token"}, scenario)
			scalar, expectedScalar := `"old-token"`, `"new-token"`
			switch scenario {
			case "plain token":
				scalar, expectedScalar = "old-token", "new-token"
			case "single quoted token":
				scalar, expectedScalar = "'old-token'", "'new-token'"
			case "flow token":
				content = "{name: café, token: \"old-token\", driver: {token: old-token}}\n"
			case "special token":
				content = "{token: \"old-token\", driver: {token: old-token}}\n"
				expectedScalar = `"new: [token], # \"quoted\"\nnext\\line"`
			}
			content = strings.Replace(content, `"old-token"`, scalar, 1)
			if scenario == "invalid config" {
				content = "token: [invalid-token]\n"
			}
			if scenario == "unsupported token" {
				content = "token: |-\n  old-token\n"
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
			newToken := "new-token"
			if scenario == "special token" {
				newToken = "new: [token], # \"quoted\"\nnext\\line"
			}
			err := m.RotateCredentials("board", strings.HasPrefix(scenario, "dry run"), scenario == "forced restart", func() (string, error) {
				rotations++
				// The preflight command must have completed before controller mutation.
				preflight := false
				for len(commands) > 0 {
					preflight = strings.Contains(<-commands, "test -w") || preflight
				}
				require.True(t, preflight, "rotation happened before SSH preflight")
				if strings.HasSuffix(scenario, "rotation fails") {
					return "", errors.New("controller unavailable")
				}
				return newToken, nil
			})
			if success || scenario == "dry run" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if success || strings.HasPrefix(scenario, "dry run") || scenario == "rotation fails" || scenario == "restart fails" {
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
			if success || scenario == "restart fails" {
				var before, after map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(content), &before))
				require.NoError(t, yaml.Unmarshal(got, &after))
				before["token"] = newToken
				require.Equal(t, before, after)
				require.Equal(t, strings.Replace(content, scalar, expectedScalar, 1), string(got))
				if scenario == "forced restart" {
					require.Equal(t, "systemctl restart -- 'board'", <-commands)
				} else {
					require.Contains(t, <-commands, "SIGHUP")
				}
			} else {
				require.Equal(t, content, string(got))
			}
			files, err := m.sftpClient.ReadDir("/etc/jumpstarter/exporters")
			require.NoError(t, err)
			require.Len(t, files, 1, "staged files must be cleaned up")
		})
	}
}

func TestExporterTokenRange(t *testing.T) {
	for _, scalar := range []string{"old-token", `"old-token"`, `'old-token'`, `"old\\\"token"`, `'old''token'`, `""`, `''`, `"old\u002dtoken"`} {
		for _, layout := range []string{
			"# old-token\ntoken: %s  # keep\ndriver:\n  token: old-token\n",
			"{name: café, token: %s, driver: {token: old-token}}\r\n",
			"'token': %s",
		} {
			content := fmt.Sprintf(layout, scalar)
			t.Run(content, func(t *testing.T) {
				token, err := exporterToken([]byte(content))
				require.NoError(t, err)
				start, end, err := exporterTokenRange([]byte(content), token)
				require.NoError(t, err)
				require.Equal(t, scalar, content[start:end])
				require.Equal(t, strings.Index(layout, "%s"), start)
			})
		}
	}
}

func TestExporterTokenRangeRejectsUnsupportedScalars(t *testing.T) {
	for _, content := range []string{
		"token: |\n  secret\n", "token: >-\n  secret\n",
		"token: secret\n  continuation\n", "token: 'secret\n  continuation'\n",
		"token: \"secret\\\n  continuation\"\n", "token: !!str secret\n",
		"token: &credential secret\nother: *credential\n",
		"other: &credential secret\ntoken: *credential\n",
		"name: host\rtoken: secret\r", "name: host\u0085token: secret\n",
		"\ufefftoken: secret\n", "name: host\u2028token: secret\n", "name: host\u2029token: secret\n",
		"\xff\xfet\x00o\x00k\x00e\x00n\x00:\x00 \x00s\x00e\x00c\x00r\x00e\x00t\x00\n\x00", // UTF-16LE
	} {
		t.Run(content, func(t *testing.T) {
			token, err := exporterToken([]byte(content))
			if err == nil {
				_, _, err = exporterTokenRange([]byte(content), token)
			}
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestExporterTokenRejectsAmbiguousConfigs(t *testing.T) {
	for _, content := range []string{"", "[]", "token: old\ntoken: other", "token: old\n---\ntoken: other", "token: [secret]", "driver: {}"} {
		_, err := exporterToken([]byte(content))
		require.Error(t, err)
		require.False(t, strings.Contains(err.Error(), "secret"), "errors must not echo config values")
	}
}
