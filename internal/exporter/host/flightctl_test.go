package host

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

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/flightctl"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
)

// fakeFlightctl serves one device and records the patches it receives.
type fakeFlightctl struct {
	device  string
	patches []string
}

func (f *fakeFlightctl) client(t *testing.T) *flightctl.Client {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			body, _ := io.ReadAll(r.Body)
			f.patches = append(f.patches, string(body))
			return
		}
		_, _ = io.WriteString(w, `{"items":[`+f.device+`]}`)
	}))
	t.Cleanup(server.Close)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	config, err := yaml.Marshal(map[string]any{
		"service":        map[string]any{"server": server.URL, "certificate-authority-data": caPEM},
		"authentication": map[string]any{"access-token": "test-token"},
	})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "client.yaml")
	require.NoError(t, os.WriteFile(path, config, 0o600))

	client, err := flightctl.NewClientFromConfigFile(path)
	require.NoError(t, err)
	return client
}

// board01 is a container exporter with the given exporter config.
func board01(t *testing.T, config string) []template.ExporterHostFiles {
	t.Helper()
	files, err := template.HostFiles(&api.ExporterConfigTemplate{Spec: api.ExporterConfigTemplateSpec{
		ExporterMetadata:         api.ExporterMeta{Name: "board-01"},
		ConfigTemplate:           config,
		SystemdContainerTemplate: "[Container]\n",
	}})
	require.NoError(t, err)
	return []template.ExporterHostFiles{files}
}

var selector = map[string]string{"host": "sidekick-01"}

func TestApplyFlightctlFilesUpdatesOnlyItsOwnProvider(t *testing.T) {
	fake := &fakeFlightctl{device: `{"metadata":{"name":"abc123","resourceVersion":"7"},"spec":{"os":{"image":"quay.io/lab/os:1"},"config":[
		{"name":"site","gitRef":{"repository":"site","path":"/lab","targetRevision":"main"}},
		{"name":"jumpstarter-exporters","inline":[
			{"path":"/etc/containers/systemd/board-01.container","content":"[Container]\n","mode":420},
			{"path":"/etc/jumpstarter/exporters/board-01.yaml","content":"token: old\n","mode":384},
			{"path":"/etc/jumpstarter/exporters/board-02.yaml","content":"dG9rZW46IG90aGVyCg==","contentEncoding":"base64",
				"mode":384,"user":"jumpstarter","group":"jumpstarter"},
			{"path":"/etc/systemd/system/board-01.service","content":"[Service]\n","mode":420}]}]}}`}
	out := NewOutputBuffer("sidekick-01", 1)

	err := applyFlightctlFiles(context.Background(), fake.client(t), selector,
		board01(t, "token: new\n"), false, out)
	require.NoError(t, err)

	require.Len(t, fake.patches, 1)
	var patch []struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	require.NoError(t, json.Unmarshal([]byte(fake.patches[0]), &patch))
	require.Len(t, patch, 2)
	assert.Equal(t, "test", patch[0].Op)
	assert.Equal(t, "/metadata/resourceVersion", patch[0].Path)
	assert.JSONEq(t, `"7"`, string(patch[0].Value))
	assert.Equal(t, "add", patch[1].Op)
	assert.Equal(t, "/spec/config", patch[1].Path)
	var config []json.RawMessage
	require.NoError(t, json.Unmarshal(patch[1].Value, &config))
	require.Len(t, config, 2)
	assert.JSONEq(t, `{"name":"site","gitRef":{"repository":"site","path":"/lab","targetRevision":"main"}}`,
		string(config[0]))
	assert.JSONEq(t, `{"name":"jumpstarter-exporters","inline":[
		{"path":"/etc/containers/systemd/board-01.container","content":"[Container]\n","mode":420},
		{"path":"/etc/jumpstarter/exporters/board-01.yaml","content":"token: new\n","mode":384},
		{"path":"/etc/jumpstarter/exporters/board-02.yaml","content":"dG9rZW46IG90aGVyCg==","contentEncoding":"base64",
			"mode":384,"user":"jumpstarter","group":"jumpstarter"}]}`,
		string(config[1]))

	output := out.buf.String()
	assert.True(t, out.hasChanges)
	assert.Contains(t, output, "Changes for file: /etc/jumpstarter/exporters/board-01.yaml")
	assert.Contains(t, output, "Removed file: /etc/systemd/system/board-01.service")
	assert.NotContains(t, output, "board-02")
	assert.NotContains(t, output, "token: new", "the diff must not show the token")
}

func TestApplyFlightctlFilesFirstDelivery(t *testing.T) {
	fake := &fakeFlightctl{device: `{"metadata":{"name":"abc123","resourceVersion":"7"}}`}
	out := NewOutputBuffer("sidekick-01", 1)

	err := applyFlightctlFiles(context.Background(), fake.client(t), selector,
		board01(t, "token: new\n"), false, out)
	require.NoError(t, err)

	require.Len(t, fake.patches, 1)
	assert.JSONEq(t, `[{"op":"test","path":"/metadata/resourceVersion","value":"7"},
		{"op":"add","path":"/spec","value":{"config":[{"name":"jumpstarter-exporters","inline":[
		{"path":"/etc/containers/systemd/board-01.container","content":"[Container]\n","mode":420},
		{"path":"/etc/jumpstarter/exporters/board-01.yaml","content":"token: new\n","mode":384}]}]}}]`, fake.patches[0])
	assert.Contains(t, out.buf.String(), "New file: /etc/jumpstarter/exporters/board-01.yaml")
}

func TestApplyFlightctlFilesWithoutChanges(t *testing.T) {
	fake := &fakeFlightctl{device: `{"metadata":{"name":"abc123"},"spec":{"config":[
		{"name":"jumpstarter-exporters","inline":[
			{"path":"/etc/containers/systemd/board-01.container","content":"[Container]\n","mode":420},
			{"path":"/etc/jumpstarter/exporters/board-01.yaml","content":"token: same\n","mode":384}]}]}}`}
	out := NewOutputBuffer("sidekick-01", 1)

	err := applyFlightctlFiles(context.Background(), fake.client(t), selector,
		board01(t, "token: same\n"), false, out)
	require.NoError(t, err)

	assert.Empty(t, fake.patches)
	assert.False(t, out.hasChanges)
}

func TestApplyFlightctlFilesDryRun(t *testing.T) {
	fake := &fakeFlightctl{device: `{"metadata":{"name":"abc123"},"spec":{}}`}
	out := NewOutputBuffer("sidekick-01", 1)

	err := applyFlightctlFiles(context.Background(), fake.client(t), selector,
		board01(t, "token: new\n"), true, out)
	require.NoError(t, err)

	assert.Empty(t, fake.patches)
	assert.True(t, out.hasChanges)
	assert.Contains(t, out.buf.String(), "Would update flightctl device abc123")
}

func TestApplyFlightctlFilesRejectsFleetOwnedDevice(t *testing.T) {
	fake := &fakeFlightctl{device: `{"metadata":{"name":"abc123","owner":"Fleet/lab"},"spec":{}}`}
	out := NewOutputBuffer("sidekick-01", 1)

	err := applyFlightctlFiles(context.Background(), fake.client(t), selector,
		board01(t, "token: new\n"), false, out)

	assert.ErrorContains(t, err, "is owned by Fleet/lab")
	assert.Empty(t, fake.patches)
}
