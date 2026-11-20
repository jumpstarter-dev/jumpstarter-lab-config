package template

import (
	"fmt"
	"io/fs"

	v1alpha1 "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
)

// HostFile is a file an exporter needs on its host. Empty Content means the file must not exist.
type HostFile struct {
	Path    string
	Content string
	// Mode is the permission mode the file needs. Only the flightctl backend applies it:
	// over SSH a file keeps the mode it was created with.
	Mode fs.FileMode
}

// ExporterHostFiles are the files that carry one exporter on its host.
type ExporterHostFiles struct {
	// ExporterName is the exporter and systemd unit name.
	ExporterName  string
	ContainerUnit HostFile
	ServiceUnit   HostFile
	Config        HostFile
}

// HostFiles returns where a rendered exporter template lives on the exporter host.
func HostFiles(exporterConfig *v1alpha1.ExporterConfigTemplate) (ExporterHostFiles, error) {
	spec := exporterConfig.Spec
	if spec.SystemdContainerTemplate != "" && spec.SystemdServiceTemplate != "" {
		return ExporterHostFiles{}, fmt.Errorf(
			"both SystemdContainerTemplate and SystemdServiceTemplate specified - only one should be used")
	}

	name := spec.ExporterMetadata.Name
	return ExporterHostFiles{
		ExporterName: name,
		ContainerUnit: HostFile{
			Path:    "/etc/containers/systemd/" + name + ".container",
			Content: spec.SystemdContainerTemplate,
			Mode:    0o644,
		},
		ServiceUnit: HostFile{
			Path:    "/etc/systemd/system/" + name + ".service",
			Content: spec.SystemdServiceTemplate,
			Mode:    0o644,
		},
		Config: HostFile{
			Path:    "/etc/jumpstarter/exporters/" + name + ".yaml",
			Content: spec.ConfigTemplate,
			// The exporter config holds the exporter token
			Mode: 0o600,
		},
	}, nil
}
