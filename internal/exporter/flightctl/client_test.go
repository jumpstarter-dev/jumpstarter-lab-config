package flightctl

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// newTestClient writes a flightctl client config for the test server and loads it.
func newTestClient(t *testing.T, server *httptest.Server, organization string) *Client {
	t.Helper()
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	config, err := yaml.Marshal(map[string]any{
		"service":        map[string]any{"server": server.URL, "certificate-authority-data": caPEM},
		"authentication": map[string]any{"access-token": "test-token", "token-to-use": "access"},
		"organization":   organization,
	})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "client.yaml")
	require.NoError(t, os.WriteFile(path, config, 0o600))

	client, err := NewClientFromConfigFile(path)
	require.NoError(t, err)
	return client
}

func TestFindDevice(t *testing.T) {
	var request *http.Request
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		_, _ = io.WriteString(w, `{"items":[{"metadata":{"name":"abc123","owner":"Fleet/lab","resourceVersion":"7"},
			"spec":{"config":[{"name":"site","inline":[]}]}}]}`)
	}))
	defer server.Close()

	device, err := newTestClient(t, server, "org-1").FindDevice(context.Background(),
		map[string]string{"host": "sidekick-01", "board": "ti"})
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/devices", request.URL.Path)
	assert.Equal(t, "board=ti,host=sidekick-01", request.URL.Query().Get("labelSelector"))
	assert.Equal(t, "org-1", request.URL.Query().Get("org_id"))
	assert.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
	assert.Equal(t, "abc123", device.Metadata.Name)
	assert.Equal(t, "Fleet/lab", device.Metadata.Owner)
	assert.Equal(t, "7", device.Metadata.ResourceVersion)
	require.Len(t, device.Spec.Config, 1)
}

func TestFindDeviceNeedsExactlyOneMatch(t *testing.T) {
	for name, body := range map[string]string{
		"none":    `{"items":[]}`,
		"several": `{"items":[{"metadata":{"name":"a"}},{"metadata":{"name":"b"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()

			_, err := newTestClient(t, server, "").FindDevice(context.Background(), map[string]string{"host": "x"})
			assert.ErrorContains(t, err, `match "host=x", expected exactly one`)
		})
	}
}

func TestSetDeviceConfig(t *testing.T) {
	config := []json.RawMessage{json.RawMessage(`{"name":"jumpstarter-exporters","inline":[]}`)}

	for name, tc := range map[string]struct {
		device *Device
		patch  string
	}{
		"device with a spec": {
			device: &Device{Metadata: DeviceMetadata{Name: "abc123", ResourceVersion: "7"}, Spec: &DeviceSpec{}},
			patch: `[{"op":"test","path":"/metadata/resourceVersion","value":"7"},
				{"op":"add","path":"/spec/config","value":[{"name":"jumpstarter-exporters","inline":[]}]}]`,
		},
		"device without a spec": {
			device: &Device{Metadata: DeviceMetadata{Name: "abc123", ResourceVersion: "7"}},
			patch: `[{"op":"test","path":"/metadata/resourceVersion","value":"7"},
				{"op":"add","path":"/spec","value":{"config":[{"name":"jumpstarter-exporters","inline":[]}]}}]`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var request *http.Request
			var body []byte
			server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				request = r
				body, _ = io.ReadAll(r.Body)
			}))
			defer server.Close()

			require.NoError(t, newTestClient(t, server, "").SetDeviceConfig(context.Background(), tc.device, config))

			assert.Equal(t, http.MethodPatch, request.Method)
			assert.Equal(t, "/api/v1/devices/abc123", request.URL.Path)
			assert.Equal(t, "application/json-patch+json", request.Header.Get("Content-Type"))
			assert.JSONEq(t, tc.patch, string(body))
		})
	}
}

func TestAPIErrorIsReported(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"updating the resource is not allowed because it has an owner"}`, http.StatusConflict)
	}))
	defer server.Close()

	err := newTestClient(t, server, "").SetDeviceConfig(context.Background(),
		&Device{Metadata: DeviceMetadata{Name: "abc123"}, Spec: &DeviceSpec{}}, nil)
	assert.ErrorContains(t, err, "409 Conflict")
	assert.ErrorContains(t, err, "it has an owner")
}

func TestClientConfigWithoutToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.yaml")
	require.NoError(t, os.WriteFile(path, []byte("service:\n  server: https://flightctl.example\n"), 0o600))

	_, err := NewClientFromConfigFile(path)
	assert.ErrorContains(t, err, "has no token")
}
