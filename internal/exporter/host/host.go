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
	"fmt"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/ssh"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
)

// SyncExporterHosts synchronizes exporter hosts via SSH
func SyncExporterHosts(cfg *config.Config, tapplier *templating.TemplateApplier, serviceParametersMap map[string]template.ServiceParameters) error {
	fmt.Print("\n🔄 Syncing exporter hosts via SSH ===========================\n\n")
	for _, host := range cfg.Loaded.ExporterHosts {
		hostCopy := host.DeepCopy()
		err := tapplier.Apply(hostCopy)
		if err != nil {
			return fmt.Errorf("error applying template for %s: %w", host.Name, err)
		}
		fmt.Printf("💻  Exporter host: %s\n", hostCopy.Name)
		hostSsh, err := ssh.NewSSHHostManager(hostCopy)
		if err != nil {
			return fmt.Errorf("error creating SSH host manager for %s: %w", host.Name, err)
		}
		status, err := hostSsh.Status()
		if err != nil {
			return fmt.Errorf("error getting status for %s: %w", host.Name, err)
		}
		fmt.Printf("    Connection status: %s\n", status)

		exporterInstances := cfg.Loaded.GetExporterInstancesByExporterHost(host.Name)
		for _, exporterInstance := range exporterInstances {
			fmt.Printf("    📟 Exporter instance: %s\n", exporterInstance.Name)
			errName := "ExporterInstance:" + exporterInstance.Name
			et, err := template.NewExporterInstanceTemplater(cfg, exporterInstance)
			serviceParameters, ok := serviceParametersMap[exporterInstance.Name]
			if !ok {
				return fmt.Errorf("service parameters not found for %s", exporterInstance.Name)
			}
			et.SetServiceParameters(serviceParameters)
			if err != nil {
				return fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
			}

			_, err = et.RenderTemplateLabels()
			if err != nil {
				return fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
			}
			_, err = et.RenderTemplateConfig()
			if err != nil {
				return fmt.Errorf("error rendering template config for %s : %w", errName, err)
			}
			fmt.Printf("   	[%s] Rendered config correctly\n", exporterInstance.Name)
		}
	}
	return nil
}
