package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UpdateAnnotationInSourceFile updates an annotation value in a YAML source file.
// It uses yaml.Node for round-trip parsing to preserve comments and formatting.
// For multi-document YAML files, it matches on metadata.name to find the correct document.
func UpdateAnnotationInSourceFile(filePath, objectName, annotationKey, newValue string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Parse all documents using yaml.Node
	var documents []*yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	preserveLeadingDocStart := hasLeadingDocStart(data)

	for {
		var doc yaml.Node
		err := decoder.Decode(&doc)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error decoding YAML from %s: %w", filePath, err)
		}
		documents = append(documents, &doc)
	}

	if len(documents) == 0 {
		return fmt.Errorf("no YAML documents found in %s", filePath)
	}

	found := false
	for _, doc := range documents {
		// Each decoded yaml.Node is a Document node wrapping the actual content
		if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
			continue
		}

		root := doc.Content[0]
		if root.Kind != yaml.MappingNode {
			continue
		}

		// Check if this document's metadata.name matches
		name := getMetadataName(root)
		if name != objectName {
			continue
		}

		// Find or create the annotations key under metadata
		if err := setAnnotationValue(root, annotationKey, newValue); err != nil {
			return fmt.Errorf("error setting annotation in %s: %w", filePath, err)
		}
		found = true
		break
	}

	if !found {
		return fmt.Errorf("object %s not found in %s", objectName, filePath)
	}

	// Write back
	out, err := marshalDocuments(documents, preserveLeadingDocStart)
	if err != nil {
		return fmt.Errorf("error marshaling YAML for %s: %w", filePath, err)
	}

	// Preserve original file permissions
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("error stating file %s: %w", filePath, err)
	}

	return writeFileAtomically(filePath, out, info.Mode())
}

// getMetadataName extracts metadata.name from a mapping node
func getMetadataName(root *yaml.Node) string {
	for i := 0; i < len(root.Content)-1; i += 2 {
		keyNode := root.Content[i]
		valNode := root.Content[i+1]

		if keyNode.Value == "metadata" && valNode.Kind == yaml.MappingNode {
			for j := 0; j < len(valNode.Content)-1; j += 2 {
				if valNode.Content[j].Value == "name" {
					return valNode.Content[j+1].Value
				}
			}
		}
	}
	return ""
}

// setAnnotationValue sets an annotation key/value in the metadata mapping.
// Creates the annotations mapping if it doesn't exist.
func setAnnotationValue(root *yaml.Node, annotationKey, newValue string) error {
	// Find the metadata mapping
	var metadataNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "metadata" && root.Content[i+1].Kind == yaml.MappingNode {
			metadataNode = root.Content[i+1]
			break
		}
	}

	if metadataNode == nil {
		return fmt.Errorf("metadata not found")
	}

	// Find the annotations mapping
	var annotationsNode *yaml.Node
	for i := 0; i < len(metadataNode.Content)-1; i += 2 {
		if metadataNode.Content[i].Value == "annotations" && metadataNode.Content[i+1].Kind == yaml.MappingNode {
			annotationsNode = metadataNode.Content[i+1]
			break
		}
	}

	if annotationsNode == nil {
		// Create annotations mapping
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "annotations", Tag: "!!str"}
		valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		metadataNode.Content = append(metadataNode.Content, keyNode, valNode)
		annotationsNode = valNode
	}

	// Find or create the annotation key
	for i := 0; i < len(annotationsNode.Content)-1; i += 2 {
		if annotationsNode.Content[i].Value == annotationKey {
			annotationsNode.Content[i+1].Value = newValue
			annotationsNode.Content[i+1].Tag = "!!str"
			annotationsNode.Content[i+1].Style = yaml.DoubleQuotedStyle
			return nil
		}
	}

	// Key not found, add it
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: annotationKey, Tag: "!!str"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: newValue, Tag: "!!str", Style: yaml.DoubleQuotedStyle}
	annotationsNode.Content = append(annotationsNode.Content, keyNode, valNode)

	return nil
}

func hasLeadingDocStart(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return strings.HasPrefix(trimmed, "---")
	}
	return false
}

// marshalDocuments serializes multiple yaml.Node documents back to bytes,
// separating them with ---
func marshalDocuments(documents []*yaml.Node, preserveLeadingDocStart bool) ([]byte, error) {
	var result []byte

	for i, doc := range documents {
		if i > 0 || preserveLeadingDocStart {
			result = append(result, []byte("---\n")...)
		}

		out, err := yaml.Marshal(doc)
		if err != nil {
			return nil, err
		}
		result = append(result, out...)
	}

	return result, nil
}

func writeFileAtomically(filePath string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("error creating temp file for %s: %w", filePath, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("error setting mode on temp file for %s: %w", filePath, err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("error writing temp file for %s: %w", filePath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("error syncing temp file for %s: %w", filePath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file for %s: %w", filePath, err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("error replacing %s with temp file: %w", filePath, err)
	}

	cleanup = false
	return nil
}
