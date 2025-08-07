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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSplitYAMLDocuments_SingleDocument(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_SingleDocumentWithTrailingNewline(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_MultipleDocuments(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key1: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key2: value2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key1: value1`

	expected2 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key2: value2`

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_MultipleDocumentsWithTrailingNewlines(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key1: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key2: value2
`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key1: value1`

	expected2 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key2: value2
`

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_EmptyDocument(t *testing.T) {
	content := ""

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 1)
	assert.Equal(t, "", documents[0])
}

func TestSplitYAMLDocuments_OnlySeparators(t *testing.T) {
	content := `---
---
---`

	documents := splitYAMLDocuments(content)

	// Should return the entire content as one document since no --- separators create meaningful splits
	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_SeparatorWithSpaces(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---   
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_SeparatorWithComment(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
--- # This is a comment
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_FourDashes(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
----
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_DashesInContent(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  # This comment has --- in it
data:
  description: "This has --- dashes in the value"
  key: value`

	documents := splitYAMLDocuments(content)

	// Should NOT split on --- in comments or string values
	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_LeadingSeparator(t *testing.T) {
	content := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_TrailingSeparator(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
---`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_EmptyDocumentsBetweenSeparators(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_WhitespaceOnlyDocument(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
   
   
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	// The function includes the whitespace-only section as a separate document
	require.Len(t, documents, 3)

	expected1 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1`

	expected2 := `   
   `

	expected3 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
	assert.Equal(t, expected3, documents[2])
}

func TestSplitYAMLDocuments_ComplexMultiDocument(t *testing.T) {
	content := `# First document
apiVersion: jumpstarter.dev/v1alpha1
kind: Client
metadata:
  name: client1
  namespace: default
spec:
  username: user1
---
# Second document with comment
apiVersion: jumpstarter.dev/v1alpha1
kind: Client
metadata:
  name: client2
  namespace: default
  annotations:
    description: "Client with --- in annotation"
spec:
  username: user2
----
# Third document with 4 dashes separator
apiVersion: meta.jumpstarter.dev/v1alpha1
kind: PhysicalLocation
metadata:
  name: location1
spec:
  description: "Test location"
`

	documents := splitYAMLDocuments(content)

	require.Len(t, documents, 3)

	expected1 := `# First document
apiVersion: jumpstarter.dev/v1alpha1
kind: Client
metadata:
  name: client1
  namespace: default
spec:
  username: user1`

	expected2 := `# Second document with comment
apiVersion: jumpstarter.dev/v1alpha1
kind: Client
metadata:
  name: client2
  namespace: default
  annotations:
    description: "Client with --- in annotation"
spec:
  username: user2`

	expected3 := `# Third document with 4 dashes separator
apiVersion: meta.jumpstarter.dev/v1alpha1
kind: PhysicalLocation
metadata:
  name: location1
spec:
  description: "Test location"
`

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
	assert.Equal(t, expected3, documents[2])
}

func TestSplitYAMLDocuments_InvalidSeparatorTwoDashes(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
--
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	// Should NOT split on -- (only two dashes), treat as single document
	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_InvalidSeparatorOneDash(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
-
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	// Should NOT split on - (only one dash), treat as single document
	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_TwoConsecutiveSeparators(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2`

	documents := splitYAMLDocuments(content)

	// Should handle consecutive separators gracefully, skipping empty sections
	require.Len(t, documents, 2)

	expected1 := testConfigMap1

	expected2 := testConfigMap2

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
}

func TestSplitYAMLDocuments_MultipleSeparatorsNoContent(t *testing.T) {
	content := `---
---
---
---`

	documents := splitYAMLDocuments(content)

	// All separators, no actual content - should return entire content as one document
	require.Len(t, documents, 1)
	assert.Equal(t, content, documents[0])
}

func TestSplitYAMLDocuments_MixedValidInvalidSeparators(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
--
this line with -- should not split
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
-
single dash should not split
----
apiVersion: v1
kind: ConfigMap
metadata:
  name: config3`

	documents := splitYAMLDocuments(content)

	// Should only split on proper --- and ---- separators
	require.Len(t, documents, 3)

	expected1 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
--
this line with -- should not split`

	expected2 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
-
single dash should not split`

	expected3 := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config3`

	assert.Equal(t, expected1, documents[0])
	assert.Equal(t, expected2, documents[1])
	assert.Equal(t, expected3, documents[2])
}
