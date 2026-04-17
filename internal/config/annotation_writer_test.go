package config

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type exporterInstanceDoc struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name        string            `yaml:"name"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
}

func decodeExporterInstanceDocs(t *testing.T, content []byte) []exporterInstanceDoc {
	t.Helper()

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	docs := make([]exporterInstanceDoc, 0)

	for {
		var doc exporterInstanceDoc
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if doc.Kind == "" && doc.Metadata.Name == "" {
			continue
		}
		docs = append(docs, doc)
	}

	return docs
}

func getExporterInstanceDocByName(docs []exporterInstanceDoc, name string) (exporterInstanceDoc, bool) {
	for _, doc := range docs {
		if doc.Metadata.Name == name {
			return doc, true
		}
	}

	return exporterInstanceDoc{}, false
}

func TestUpdateAnnotationInSourceFile_SingleDocument(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: my-exporter
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporter.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "my-exporter", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	assert.Contains(t, string(result), `jumpstarter.dev/unmanaged: "2026-02-09"`)
	// Verify other content is preserved
	assert.Contains(t, string(result), "kind: ExporterInstance")
	assert.Contains(t, string(result), "name: my-exporter")
	assert.Contains(t, string(result), "type: test")
}

func TestUpdateAnnotationInSourceFile_MultiDocument(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-1
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test1
---
apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-2
spec:
  type: test2
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporters.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "exporter-1", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	docs := decodeExporterInstanceDocs(t, result)
	require.Len(t, docs, 2)

	exporter1, found := getExporterInstanceDocByName(docs, "exporter-1")
	require.True(t, found)
	assert.Equal(t, "2026-02-09", exporter1.Metadata.Annotations["jumpstarter.dev/unmanaged"])

	// Verify the second document has no annotations.
	exporter2, found := getExporterInstanceDocByName(docs, "exporter-2")
	require.True(t, found)
	assert.Empty(t, exporter2.Metadata.Annotations)
}

func TestUpdateAnnotationInSourceFile_NoAnnotationsSection(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: my-exporter
spec:
  type: test
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporter.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "my-exporter", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	resultStr := string(result)
	assert.Contains(t, resultStr, "annotations:")
	assert.Contains(t, resultStr, `jumpstarter.dev/unmanaged: "2026-02-09"`)
}

func TestUpdateAnnotationInSourceFile_ObjectNotFound(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: other-exporter
spec:
  type: test
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporter.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "my-exporter", "jumpstarter.dev/unmanaged", "2026-02-09")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object my-exporter not found")
}

func TestUpdateAnnotationInSourceFile_PreservesComments(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: my-exporter
  annotations:
    # This exporter is being debugged
    jumpstarter.dev/unmanaged: ""
spec:
  type: test # important type
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporter.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "my-exporter", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	resultStr := string(result)
	assert.Contains(t, resultStr, `jumpstarter.dev/unmanaged: "2026-02-09"`)
	assert.Contains(t, resultStr, "This exporter is being debugged")
}

func TestUpdateAnnotationInSourceFile_UpdatesCorrectDocumentInMultiDoc(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-1
spec:
  type: test1
---
apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-2
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test2
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporters.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	// Update annotation on second document
	err := UpdateAnnotationInSourceFile(filePath, "exporter-2", "jumpstarter.dev/unmanaged", "2026-01-15")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	docs := decodeExporterInstanceDocs(t, result)
	require.Len(t, docs, 2)

	exporter2, found := getExporterInstanceDocByName(docs, "exporter-2")
	require.True(t, found)
	assert.Equal(t, "2026-01-15", exporter2.Metadata.Annotations["jumpstarter.dev/unmanaged"])

	exporter1, found := getExporterInstanceDocByName(docs, "exporter-1")
	require.True(t, found)
	assert.Empty(t, exporter1.Metadata.Annotations)
}

func TestUpdateAnnotationInSourceFile_DecodeErrorIsReturned(t *testing.T) {
	content := `apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: my-exporter
spec:
  type: test
---
apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: broken-exporter
spec:
  type: [invalid
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporters.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "my-exporter", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error decoding YAML")
}

func TestUpdateAnnotationInSourceFile_PreservesLeadingDocStart(t *testing.T) {
	content := `---
apiVersion: jumpstarter.dev/v1alpha1
kind: ExporterInstance
metadata:
  name: exporter-1
  annotations:
    jumpstarter.dev/unmanaged: ""
spec:
  type: test
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exporter.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := UpdateAnnotationInSourceFile(filePath, "exporter-1", "jumpstarter.dev/unmanaged", "2026-02-09")
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(result), "---\n"))
}
