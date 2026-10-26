/*
Copyright 2025. The Jumpstarter Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testConfigMap1 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1`

	testConfigMap2 = `apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`
)

// splitYAMLDocuments delegates to apimachinery's YAMLReader, which only treats
// `---` at the very start of a line as a document separator. The reader also
// normalizes line endings and appends a trailing newline to each returned
// document, so content assertions trim that cosmetic whitespace.

func TestSplitYAMLDocuments_SingleDocument(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value`

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, content, strings.TrimRight(docs[0], "\n"))
}

func TestSplitYAMLDocuments_SingleDocumentWithTrailingNewline(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, strings.TrimSpace(content), strings.TrimSpace(docs[0]))
}

func TestSplitYAMLDocuments_MultipleDocuments(t *testing.T) {
	content := testConfigMap1 + "\n---\n" + testConfigMap2

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, testConfigMap1, strings.TrimSpace(docs[0]))
	assert.Equal(t, testConfigMap2, strings.TrimSpace(docs[1]))
}

func TestSplitYAMLDocuments_EmptyContent(t *testing.T) {
	docs, err := splitYAMLDocuments("")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestSplitYAMLDocuments_SeparatorWithTrailingSpaces(t *testing.T) {
	content := testConfigMap1 + "\n---   \n" + testConfigMap2

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, testConfigMap1, strings.TrimSpace(docs[0]))
	assert.Equal(t, testConfigMap2, strings.TrimSpace(docs[1]))
}

func TestSplitYAMLDocuments_SeparatorWithComment(t *testing.T) {
	content := testConfigMap1 + "\n--- # This is a comment\n" + testConfigMap2

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, testConfigMap1, strings.TrimSpace(docs[0]))
	assert.Equal(t, testConfigMap2, strings.TrimSpace(docs[1]))
}

// TestSplitYAMLDocuments_IndentedSeparatorInBlockScalar is the regression test
// for the reported bug: an indented `---` inside a `notes: |` block scalar (a
// markdown horizontal rule) must NOT start a new document.
func TestSplitYAMLDocuments_IndentedSeparatorInBlockScalar(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  notes: |
    intro
    ---
    outro`

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 1, "an indented --- inside a block scalar must not split the document")
	assert.Equal(t, content, strings.TrimRight(docs[0], "\n"))
}

// TestSplitYAMLDocuments_PEMHeaderInBlockScalar guards the other reported case:
// a `-----BEGIN` PEM line inside a block scalar must not be treated as a
// separator.
func TestSplitYAMLDocuments_PEMHeaderInBlockScalar(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  cert: |
    -----BEGIN CERTIFICATE-----
    abcdef
    -----END CERTIFICATE-----`

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, content, strings.TrimRight(docs[0], "\n"))
}

func TestSplitYAMLDocuments_DashesInCommentOrValue(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  # This comment has --- in it
data:
  description: "This has --- dashes in the value"
  key: value`

	docs, err := splitYAMLDocuments(content)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, content, strings.TrimRight(docs[0], "\n"))
}

func TestSplitYAMLDocuments_OneOrTwoDashesIsNotASeparator(t *testing.T) {
	for _, sep := range []string{"-", "--"} {
		content := testConfigMap1 + "\n" + sep + "\n" + testConfigMap2
		docs, err := splitYAMLDocuments(content)
		require.NoError(t, err)
		assert.Len(t, docs, 1, "%q should not be treated as a document separator", sep)
	}
}

// TestSplitYAMLDocuments_FourDashesIsInvalid documents a deliberate behavior
// change from the old hand-rolled splitter: `----` is not a valid YAML document
// separator and apimachinery's reader rejects it.
func TestSplitYAMLDocuments_FourDashesIsInvalid(t *testing.T) {
	content := testConfigMap1 + "\n----\n" + testConfigMap2

	_, err := splitYAMLDocuments(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "separator")
}

// ---- readAndDecodeYAMLFile integration tests ----

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resource.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// TestReadAndDecodeYAMLFile_BlockScalarWithMarkdownRule is the end-to-end
// regression for the reported failure: a file with a markdown horizontal rule
// inside a block scalar used to be split into two documents and fail to load.
func TestReadAndDecodeYAMLFile_BlockScalarWithMarkdownRule(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: with-notes
data:
  notes: |
    # Heading

    some text
    ---
    more text after a horizontal rule
`

	objs, err := readAndDecodeYAMLFile(writeTempYAML(t, content))
	require.NoError(t, err)
	require.Len(t, objs, 1)

	cm, ok := objs[0].(*corev1.ConfigMap)
	require.True(t, ok, "expected a *corev1.ConfigMap, got %T", objs[0])
	assert.Equal(t, "with-notes", cm.Name)
	assert.Contains(t, cm.Data["notes"], "---")
	assert.Contains(t, cm.Data["notes"], "more text after a horizontal rule")
}

func TestReadAndDecodeYAMLFile_PEMBlock(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: with-cert
data:
  tls.crt: |
    -----BEGIN CERTIFICATE-----
    MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
    -----END CERTIFICATE-----
`

	objs, err := readAndDecodeYAMLFile(writeTempYAML(t, content))
	require.NoError(t, err)
	require.Len(t, objs, 1)

	cm, ok := objs[0].(*corev1.ConfigMap)
	require.True(t, ok, "expected a *corev1.ConfigMap, got %T", objs[0])
	assert.Contains(t, cm.Data["tls.crt"], "-----BEGIN CERTIFICATE-----")
	assert.Contains(t, cm.Data["tls.crt"], "-----END CERTIFICATE-----")
}

// TestReadAndDecodeYAMLFile_MultiDocument also exercises leading, trailing and
// blank separator sections, all of which must be handled gracefully.
func TestReadAndDecodeYAMLFile_MultiDocument(t *testing.T) {
	content := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
---
`

	objs, err := readAndDecodeYAMLFile(writeTempYAML(t, content))
	require.NoError(t, err)
	require.Len(t, objs, 2)

	names := make([]string, 0, len(objs))
	for _, obj := range objs {
		cm, ok := obj.(*corev1.ConfigMap)
		require.True(t, ok, "expected a *corev1.ConfigMap, got %T", obj)
		names = append(names, cm.Name)
	}
	assert.ElementsMatch(t, []string{"config1", "config2"}, names)
}

func TestReadAndDecodeYAMLFile_SingleDocument(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: solo
data:
  key: value`

	objs, err := readAndDecodeYAMLFile(writeTempYAML(t, content))
	require.NoError(t, err)
	require.Len(t, objs, 1)

	cm, ok := objs[0].(*corev1.ConfigMap)
	require.True(t, ok, "expected a *corev1.ConfigMap, got %T", objs[0])
	assert.Equal(t, "solo", cm.Name)
	assert.Equal(t, "value", cm.Data["key"])
}
