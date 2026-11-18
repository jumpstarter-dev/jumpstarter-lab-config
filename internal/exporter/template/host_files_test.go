package template

import (
	"testing"

	v1alpha1 "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostFiles(t *testing.T) {
	files, err := HostFiles(&v1alpha1.ExporterConfigTemplate{
		Spec: v1alpha1.ExporterConfigTemplateSpec{
			ExporterMetadata:         v1alpha1.ExporterMeta{Name: "board-01"},
			ConfigTemplate:           "endpoint: example",
			SystemdContainerTemplate: "[Container]",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, ExporterHostFiles{
		ExporterName:  "board-01",
		ContainerUnit: HostFile{Path: "/etc/containers/systemd/board-01.container", Content: "[Container]", Mode: 0o644},
		ServiceUnit:   HostFile{Path: "/etc/systemd/system/board-01.service", Mode: 0o644},
		Config:        HostFile{Path: "/etc/jumpstarter/exporters/board-01.yaml", Content: "endpoint: example", Mode: 0o600},
	}, files)
}

func TestHostFilesRejectsBothUnitTemplates(t *testing.T) {
	_, err := HostFiles(&v1alpha1.ExporterConfigTemplate{
		Spec: v1alpha1.ExporterConfigTemplateSpec{
			ExporterMetadata:         v1alpha1.ExporterMeta{Name: "board-01"},
			SystemdContainerTemplate: "[Container]",
			SystemdServiceTemplate:   "[Service]",
		},
	})
	assert.ErrorContains(t, err, "only one should be used")
}
