package flightctl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeFilesKeepsFilesOfOtherExporters(t *testing.T) {
	current := []File{
		{Path: "/etc/jumpstarter/exporters/b.yaml", Content: "b-old"},
		{Path: "/etc/jumpstarter/exporters/a.yaml", Content: "a"},
		{Path: "/etc/systemd/system/b.service", Content: "unit"},
	}

	merged := MergeFiles(current,
		[]File{{Path: "/etc/jumpstarter/exporters/b.yaml", Content: "b-new"}},
		[]string{"/etc/systemd/system/b.service"})

	assert.Equal(t, []File{
		{Path: "/etc/jumpstarter/exporters/a.yaml", Content: "a"},
		{Path: "/etc/jumpstarter/exporters/b.yaml", Content: "b-new"},
	}, merged)
}

func TestSplitAndJoinConfigLeaveOtherProvidersUntouched(t *testing.T) {
	other := json.RawMessage(`{"name":"site","gitRef":{"repository":"site","path":"/lab","targetRevision":"main"}}`)
	ours := json.RawMessage(`{"name":"jumpstarter-exporters","inline":[{"path":"/old","content":"old","mode":384}]}`)

	files, others, err := SplitConfig([]json.RawMessage{ours, other})
	require.NoError(t, err)
	assert.Equal(t, []File{{Path: "/old", Content: "old", Mode: 0o600}}, files)
	assert.Equal(t, []json.RawMessage{other}, others)

	config, err := JoinConfig(others, []File{{Path: "/new", Content: "new"}})
	require.NoError(t, err)
	require.Len(t, config, 2)
	assert.Equal(t, other, config[0])
	assert.JSONEq(t, `{"name":"jumpstarter-exporters","inline":[{"path":"/new","content":"new"}]}`, string(config[1]))
}

func TestSplitConfigWithoutOurProvider(t *testing.T) {
	other := json.RawMessage(`{"name":"site","inline":[]}`)

	files, others, err := SplitConfig([]json.RawMessage{other})
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Equal(t, []json.RawMessage{other}, others)
}
