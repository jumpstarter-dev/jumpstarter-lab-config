package unmanaged

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fixedNow() time.Time {
	return time.Date(2026, time.February, 17, 12, 0, 0, 0, time.UTC)
}

func TestProcessUnmanagedExporters_NoUnmanaged(t *testing.T) {
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.Empty(t, result.Exporters)
}

func TestProcessUnmanagedExporters_WithDate(t *testing.T) {
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "2026-01-01",
						},
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])
	assert.Len(t, result.Exporters, 1)
}

func TestProcessUnmanagedExporters_EmptyValueSetsDate(t *testing.T) {
	// Create a temp YAML file for write-back
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "exporter.yaml")
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-1
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test
`
	require.NoError(t, os.WriteFile(yamlFile, []byte(content), 0644))

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "",
						},
					},
				},
			},
			SourceFiles: map[string]map[string]string{
				"ExporterInstance": {
					"exporter-1": yamlFile,
				},
			},
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])

	// Verify in-memory annotation was updated
	today := fixedNow().Format("2006-01-02")
	assert.Equal(t, today, cfg.Loaded.ExporterInstances["exporter-1"].Annotations[api.UnmanagedAnnotation])

	// Verify file was updated
	fileContent, err := os.ReadFile(yamlFile)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), today)
}

func TestProcessUnmanagedExporters_EmptyValueWriteFailureDoesNotUpdateInMemory(t *testing.T) {
	tmpDir := t.TempDir()
	missingFile := filepath.Join(tmpDir, "missing-exporter.yaml")

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "",
						},
					},
				},
			},
			SourceFiles: map[string]map[string]string{
				"ExporterInstance": {
					"exporter-1": missingFile,
				},
			},
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])
	assert.Equal(t, "", cfg.Loaded.ExporterInstances["exporter-1"].Annotations[api.UnmanagedAnnotation])
}

func TestProcessUnmanagedExporters_InvalidDate(t *testing.T) {
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "not-a-date",
						},
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	// Should still be treated as unmanaged
	assert.True(t, result.Exporters["exporter-1"])
}

func TestProcessUnmanagedExporters_FutureDate(t *testing.T) {
	futureDate := fixedNow().AddDate(0, 0, 7).Format("2006-01-02")
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: futureDate,
						},
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])
	assert.Equal(t, 0, result.OldestDays)
}

func TestProcessUnmanagedExporters_MixedExporters(t *testing.T) {
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"managed-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "managed-1",
					},
				},
				"unmanaged-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "unmanaged-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "2026-02-01",
						},
					},
				},
				"managed-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "managed-2",
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.Len(t, result.Exporters, 1)
	assert.True(t, result.Exporters["unmanaged-1"])
	assert.NotContains(t, result.Exporters, "managed-1")
	assert.NotContains(t, result.Exporters, "managed-2")
}

func TestProcessUnmanagedExporters_DeadAndUnmanaged(t *testing.T) {
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.DeadAnnotation:      "broken hardware",
							api.UnmanagedAnnotation: "2026-01-15",
						},
					},
				},
			},
			SourceFiles: make(map[string]map[string]string),
		},
	}

	result := ProcessUnmanagedExporters(cfg, false, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])
}

func TestProcessUnmanagedExporters_EmptyValueDryRunDoesNotWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "exporter.yaml")
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-1
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test
`
	require.NoError(t, os.WriteFile(yamlFile, []byte(content), 0644))

	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			ExporterInstances: map[string]*api.ExporterInstance{
				"exporter-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "exporter-1",
						Annotations: map[string]string{
							api.UnmanagedAnnotation: "",
						},
					},
				},
			},
			SourceFiles: map[string]map[string]string{
				"ExporterInstance": {
					"exporter-1": yamlFile,
				},
			},
		},
	}

	result := ProcessUnmanagedExporters(cfg, true, fixedNow)
	assert.True(t, result.Exporters["exporter-1"])

	fileContent, err := os.ReadFile(yamlFile)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), `jumpstarter.dev/unmanaged: ""`)
}
