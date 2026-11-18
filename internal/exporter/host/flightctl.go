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

package host

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/flightctl"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/ssh"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
)

// newFlightctlClient builds a flightctl client from the client config written by "flightctl login",
// read from --flightctl-config or from where the flightctl CLI keeps it.
func (e *ExporterHostSyncer) newFlightctlClient() (*flightctl.Client, error) {
	configPath := e.flightctlConfigPath
	if configPath == "" {
		defaultPath, err := flightctl.DefaultConfigPath()
		if err != nil {
			return nil, err
		}
		configPath = defaultPath
	}
	return flightctl.NewClientFromConfigFile(configPath)
}

// syncFlightctlHost renders the exporters of a host and hands their files to flightctl.
// The host is updated in one step, so one exporter that fails to render holds back the whole host.
func (e *ExporterHostSyncer) syncFlightctlHost(
	flightctlClient func() (*flightctl.Client, error), w hostWork, out *OutputBuffer,
) error {
	client, err := flightctlClient()
	if err != nil {
		return err
	}

	exporters := make([]template.ExporterHostFiles, 0, len(w.instances))
	for _, exporterInstance := range w.instances {
		tcfg, err := e.renderExporterInstance(exporterInstance, out)
		if err != nil {
			return err
		}
		files, err := template.HostFiles(tcfg)
		if err != nil {
			return fmt.Errorf("ExporterInstance:%s: %w", exporterInstance.Name, err)
		}
		exporters = append(exporters, files)
	}

	selector := w.host.Spec.Management.FlightCtl.DeviceSelector
	return applyFlightctlFiles(context.Background(), client, selector, exporters, e.dryRun, out)
}

// applyFlightctlFiles puts the files of the given exporters into the spec of the selected device.
// It does not restart anything: the flightctl agent writes the files, and the hooks on the host
// decide when it is safe to restart an exporter.
func applyFlightctlFiles(
	ctx context.Context,
	client *flightctl.Client,
	selector map[string]string,
	exporters []template.ExporterHostFiles,
	dryRun bool,
	out *OutputBuffer,
) error {
	device, err := client.FindDevice(ctx, selector)
	if err != nil {
		return err
	}
	name := device.Metadata.Name
	if device.Metadata.Owner != "" {
		return fmt.Errorf("flightctl device %s is owned by %s: flightctl takes the spec of such a device "+
			"from the fleet template and rejects changes to the device", name, device.Metadata.Owner)
	}

	current, others, err := flightctl.SplitConfig(device.Config())
	if err != nil {
		return err
	}

	var set []flightctl.File
	var remove []string
	for _, exporter := range exporters {
		for _, file := range []template.HostFile{exporter.ContainerUnit, exporter.ServiceUnit, exporter.Config} {
			if file.Content == "" {
				remove = append(remove, file.Path)
				continue
			}
			set = append(set, flightctl.File{Path: file.Path, Content: file.Content, Mode: int(file.Mode)})
		}
	}
	desired := flightctl.MergeFiles(current, set, remove)

	if !printFileChanges(current, desired, out) {
		return nil
	}
	out.MarkChanged()

	if dryRun {
		out.Printf("        📄 Would update flightctl device %s\n", name)
		return nil
	}

	config, err := flightctl.JoinConfig(others, desired)
	if err != nil {
		return err
	}
	if err := client.SetDeviceConfig(ctx, device, config); err != nil {
		return err
	}
	out.Printf("        ✅ flightctl device %s updated, its agent applies the change on its next sync\n", name)
	return nil
}

// printFileChanges describes how desired differs from current and reports whether it does.
func printFileChanges(current, desired []flightctl.File, out *OutputBuffer) bool {
	currentByPath := make(map[string]flightctl.File, len(current))
	for _, file := range current {
		currentByPath[file.Path] = file
	}

	changed := false
	for _, file := range desired {
		existing, exists := currentByPath[file.Path]
		delete(currentByPath, file.Path)
		switch {
		case !exists:
			out.Printf("            📄 New file: %s\n", file.Path)
			changed = true
		case !cmp.Equal(existing, file):
			out.Printf("            ✏️ Changes for file: %s\n", file.Path)
			// Empty when only a redacted value or file metadata changed
			if diff := ssh.ContentDiff(existing.Content, file.Content); diff != "" {
				out.Printf("            Diff (-existing +new):\n%s\n", diff)
			}
			changed = true
		}
	}
	// What is left of current is no longer desired
	for _, file := range current {
		if _, removed := currentByPath[file.Path]; removed {
			out.Printf("            🗑️ Removed file: %s\n", file.Path)
			changed = true
		}
	}
	return changed
}
