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

package flightctl

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ProviderName names the inline config provider that holds the exporter files in a device spec.
// Every other config provider of the device is left untouched.
const ProviderName = "jumpstarter-exporters"

// File is one file of an inline config provider. It carries every field of a flightctl FileSpec,
// so that the files of exporters a run does not process are written back exactly as they were.
type File struct {
	Path            string `json:"path"`
	Content         string `json:"content"`
	ContentEncoding string `json:"contentEncoding,omitempty"`
	// Mode is the permission mode of the file. flightctl writes 0644 when it is not set.
	Mode  int    `json:"mode,omitempty"`
	User  string `json:"user,omitempty"`
	Group string `json:"group,omitempty"`
}

type inlineProvider struct {
	Name   string `json:"name"`
	Inline []File `json:"inline"`
}

// SplitConfig separates the exporter files in a device's config providers from the other providers.
func SplitConfig(config []json.RawMessage) (files []File, others []json.RawMessage, err error) {
	for _, raw := range config {
		var provider inlineProvider
		if err := json.Unmarshal(raw, &provider); err != nil {
			return nil, nil, fmt.Errorf("decoding device config provider: %w", err)
		}
		if provider.Name == ProviderName {
			files = provider.Inline
			continue
		}
		others = append(others, raw)
	}
	return files, others, nil
}

// JoinConfig returns the config providers others followed by the provider of the exporter files.
func JoinConfig(others []json.RawMessage, files []File) ([]json.RawMessage, error) {
	ours, err := json.Marshal(inlineProvider{Name: ProviderName, Inline: files})
	if err != nil {
		return nil, fmt.Errorf("encoding device config provider: %w", err)
	}
	return append(others, ours), nil
}

// MergeFiles applies set and remove to current, by path, and returns the result sorted by path.
// Files that are in neither stay as they are: they belong to exporters this run does not process.
func MergeFiles(current, set []File, remove []string) []File {
	byPath := make(map[string]File, len(current)+len(set))
	for _, file := range current {
		byPath[file.Path] = file
	}
	for _, file := range set {
		byPath[file.Path] = file
	}
	for _, path := range remove {
		delete(byPath, path)
	}

	merged := make([]File, 0, len(byPath))
	for _, file := range byPath {
		merged = append(merged, file)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Path < merged[j].Path })
	return merged
}
