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
	"regexp"
	"strings"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/ssh"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
)

type ExporterHostSyncer struct {
	cfg                  *config.Config
	tapplier             *templating.TemplateApplier
	serviceParametersMap map[string]template.ServiceParameters
	dryRun               bool
	debugConfigs         bool
	exporterFilter       *regexp.Regexp
}

func NewExporterHostSyncer(cfg *config.Config,
	tapplier *templating.TemplateApplier,
	serviceParametersMap map[string]template.ServiceParameters,
	dryRun, debugConfigs bool,
	exporterFilter *regexp.Regexp) *ExporterHostSyncer {
	return &ExporterHostSyncer{
		cfg:                  cfg,
		tapplier:             tapplier,
		serviceParametersMap: serviceParametersMap,
		dryRun:               dryRun,
		debugConfigs:         debugConfigs,
		exporterFilter:       exporterFilter,
	}
}

// isExporterInstanceDead checks if an exporter instance is marked as dead via annotation
func isExporterInstanceDead(instance *api.ExporterInstance) (bool, string) {
	if deadAnnotation, exists := instance.Annotations["dead"]; exists {
		return true, deadAnnotation
	}
	return false, ""
}

// filterExporterInstances filters instances, checks if all are dead, and handles skip printing
func (e *ExporterHostSyncer) filterExporterInstances(hostName string, exporterInstances []*api.ExporterInstance) []*api.ExporterInstance {
	// Filter instances based on regex if provided
	if e.exporterFilter != nil {
		filteredInstances := make([]*api.ExporterInstance, 0, len(exporterInstances))
		for _, exporterInstance := range exporterInstances {
			if e.exporterFilter.MatchString(exporterInstance.Name) {
				filteredInstances = append(filteredInstances, exporterInstance)
			}
		}
		exporterInstances = filteredInstances
	}

	// no instances match the filter
	if len(exporterInstances) == 0 {
		return nil
	}

	// Check if all remaining instances are dead
	allDead := true
	var deadAnnotations []string

	for _, exporterInstance := range exporterInstances {
		if isDead, deadAnnotation := isExporterInstanceDead(exporterInstance); isDead {
			deadAnnotations = append(deadAnnotations, fmt.Sprintf("%s: %s", exporterInstance.Name, deadAnnotation))
		} else {
			allDead = false
			break
		}
	}

	// all instances are dead
	if allDead {
		fmt.Printf("\n💻  Exporter host: %s skipped - all instances dead: [%s]\n", hostName, strings.Join(deadAnnotations, ", "))
		return nil
	}

	return exporterInstances
}

// handleBootcUpgrade handles bootc upgrade checking and execution
func (e *ExporterHostSyncer) handleBootcUpgrade(hostSsh ssh.HostManager) (bool, error) {
	// Check if bootc upgrade service is already running
	serviceStatus, _ := hostSsh.RunHostCommand("systemctl is-active bootc-fetch-apply-updates.service")
	if serviceStatus != nil {
		status := strings.TrimSpace(serviceStatus.Stdout)
		if status == "active" || status == "activating" {
			fmt.Printf("    ⚠️  Bootc upgrade in progress, skipping exporter instances for this host\n")
			return true, nil // skip = true
		}
	}

	// Check booted image
	bootcStdout, err := hostSsh.RunHostCommand("[ -f /run/ostree-booted ] && bootc upgrade --check")
	if err == nil && bootcStdout != nil && bootcStdout.ExitCode == 0 && bootcStdout.Stdout != "" {
		if strings.HasPrefix(bootcStdout.Stdout, "No changes") {
			if e.dryRun {
				fmt.Printf("    ✅ Bootc image is up to date\n")
			}
		} else if e.dryRun {
			fmt.Printf("    📄 Would upgrade bootc image\n")
		} else {
			// Trigger bootc upgrade service now
			_, err := hostSsh.RunHostCommand("systemctl start --no-block bootc-fetch-apply-updates.service")
			if err != nil {
				return false, fmt.Errorf("error triggering bootc upgrade service: %w", err)
			}
			fmt.Printf("    ✅ Bootc upgrade started, skipping exporter instances for this host\n")
			return true, nil // skip = true
		}
	} else {
		fmt.Printf("    ℹ️ Not a bootc managed host\n")
	}
	return false, nil // skip = false
}

// processExporterInstance processes a single exporter instance
func (e *ExporterHostSyncer) processExporterInstance(exporterInstance *api.ExporterInstance, hostSsh ssh.HostManager) error {
	if isDead, deadAnnotation := isExporterInstanceDead(exporterInstance); isDead {
		fmt.Printf("    📟 Exporter instance: %s skipped - dead: %s\n", exporterInstance.Name, deadAnnotation)
		return nil
	}

	fmt.Printf("    📟 Exporter instance: %s\n", exporterInstance.Name)
	errName := "ExporterInstance:" + exporterInstance.Name

	et, err := template.NewExporterInstanceTemplater(e.cfg, exporterInstance)
	if err != nil {
		return fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
	}

	serviceParameters, ok := e.serviceParametersMap[exporterInstance.Name]
	if !ok {
		return fmt.Errorf("service parameters not found for %s", exporterInstance.Name)
	}
	et.SetServiceParameters(serviceParameters)

	_, err = et.RenderTemplateLabels()
	if err != nil {
		return fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
	}

	tcfg, err := et.RenderTemplateConfig()
	if err != nil {
		return fmt.Errorf("error rendering template config for %s : %w", errName, err)
	}

	if e.debugConfigs {
		fmt.Printf("--- 📄 Config Template %s\n", strings.Repeat("─", 40))
		fmt.Printf("%s\n", tcfg.Spec.ConfigTemplate)
		if tcfg.Spec.SystemdContainerTemplate != "" {
			fmt.Printf("  - ⚙️  Systemd Container Template %s\n", strings.Repeat("─", 30))
			fmt.Printf("%s\n", tcfg.Spec.SystemdContainerTemplate)
		}
		if tcfg.Spec.SystemdServiceTemplate != "" {
			fmt.Printf("  - 🔧 Systemd Service Template %s\n", strings.Repeat("─", 31))
			fmt.Printf("%s\n", tcfg.Spec.SystemdServiceTemplate)
		}
		fmt.Println(strings.Repeat("─", 60))
	}

	return hostSsh.Apply(tcfg, e.dryRun)
}

// SyncExporterHosts synchronizes exporter hosts via SSH
func (e *ExporterHostSyncer) SyncExporterHosts() error {
	fmt.Print("\n🔄 Syncing exporter hosts via SSH ===========================\n")

	for _, host := range e.cfg.Loaded.ExporterHosts {
		exporterInstances := e.cfg.Loaded.GetExporterInstancesByExporterHost(host.Name)
		exporterInstances = e.filterExporterInstances(host.Name, exporterInstances)

		// Skip the host if no viable exporter instances remain
		if len(exporterInstances) == 0 {
			continue
		}

		hostCopy := host.DeepCopy()
		if err := e.tapplier.Apply(hostCopy); err != nil {
			return fmt.Errorf("error applying template for %s: %w", host.Name, err)
		}

		fmt.Printf("\n💻  Exporter host: %s\n", hostCopy.Spec.Addresses[0])

		hostSsh, err := ssh.NewSSHHostManager(hostCopy)
		if err != nil {
			return fmt.Errorf("error creating SSH host manager for %s: %w", host.Name, err)
		}

		status, err := hostSsh.Status()
		if err != nil {
			return fmt.Errorf("error getting status for %s: %w", host.Name, err)
		}
		if e.dryRun {
			fmt.Printf("    ✅ Connection: %s\n", status)
		}

		if skip, err := e.handleBootcUpgrade(hostSsh); err != nil {
			return err
		} else if skip {
			continue
		}

		// Process each exporter instance
		for _, exporterInstance := range exporterInstances {
			if err := e.processExporterInstance(exporterInstance, hostSsh); err != nil {
				return fmt.Errorf("error applying exporter config for %s: %w", exporterInstance.Name, err)
			}
		}
	}

	return nil
}
