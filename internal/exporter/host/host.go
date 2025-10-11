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
	"math"
	"regexp"
	"strings"
	"time"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/ssh"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
)

// RetryItem represents a failed exporter instance that needs to be retried
type RetryItem struct {
	ExporterInstance *api.ExporterInstance
	HostName         string
	RenderedHost     *api.ExporterHost // The rendered host with templates applied
	Attempts         int
	LastError        error
	LastAttemptTime  time.Time
}

// RetryConfig holds configuration for the retry mechanism
type RetryConfig struct {
	MaxAttempts       int           // Maximum number of retry attempts
	BaseDelay         time.Duration // Base delay for exponential backoff
	MaxDelay          time.Duration // Maximum delay cap
	BackoffMultiplier float64       // Multiplier for exponential backoff
}

type ExporterHostSyncer struct {
	cfg                  *config.Config
	tapplier             *templating.TemplateApplier
	serviceParametersMap map[string]template.ServiceParameters
	dryRun               bool
	debugConfigs         bool
	exporterFilter       *regexp.Regexp
	retryConfig          RetryConfig
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
		// this provides 10 minutes of retries with a max delay of 120 seconds
		retryConfig: RetryConfig{
			MaxAttempts:       9,
			BaseDelay:         5 * time.Second,
			MaxDelay:          120 * time.Second,
			BackoffMultiplier: 2.0,
		},
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

// processExporterInstance processes a single exporter instance
func (e *ExporterHostSyncer) processExporterInstance(exporterInstance *api.ExporterInstance, hostSsh ssh.HostManager, renderedHost *api.ExporterHost) error {
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

	spRef := exporterInstance.Spec.JumpstarterInstanceRef.Name + ":" + exporterInstance.Name
	serviceParameters, ok := e.serviceParametersMap[spRef]
	if !ok {
		return fmt.Errorf("service parameters not found for %s", spRef)
	}
	et.SetServiceParameters(serviceParameters)
	et.SetRenderedExporterHost(renderedHost)

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

// calculateBackoffDelay calculates the delay for exponential backoff
func (e *ExporterHostSyncer) calculateBackoffDelay(attempts int) time.Duration {
	delay := time.Duration(float64(e.retryConfig.BaseDelay) * math.Pow(e.retryConfig.BackoffMultiplier, float64(attempts)))
	if delay > e.retryConfig.MaxDelay {
		delay = e.retryConfig.MaxDelay
	}
	return delay
}

// addToRetryQueue increments attempts and adds a retry item to the next retry queue
func (e *ExporterHostSyncer) addToRetryQueue(retryItem *RetryItem, err error, nextRetryQueue *[]RetryItem) {
	retryItem.Attempts++
	retryItem.LastError = err
	retryItem.LastAttemptTime = time.Now()
	*nextRetryQueue = append(*nextRetryQueue, *retryItem)
}

// getRetryItemDescription returns a human-readable description of what is being retried
func getRetryItemDescription(retryItem RetryItem) string {
	if retryItem.ExporterInstance == nil {
		return "bootc upgrade"
	} else {
		return fmt.Sprintf("instance %s", retryItem.ExporterInstance.Name)
	}
}

// processExporterInstancesAndBootc processes exporter instances and adds failures to global retry queue
func (e *ExporterHostSyncer) processExporterInstancesAndBootc(exporterInstances []*api.ExporterInstance, hostName string, renderedHost *api.ExporterHost, retryQueue *[]RetryItem) {
	// Create SSH connection
	hostSsh, err := ssh.NewSSHHostManager(renderedHost)
	if err == nil {
		_, err = hostSsh.Status()
	}
	if err != nil {
		fmt.Printf("    ❌ Failed to create/test SSH connection: %v\n", err)
		// Queue all exporter instances for retry
		for _, exporterInstance := range exporterInstances {
			*retryQueue = append(*retryQueue, RetryItem{
				ExporterInstance: exporterInstance,
				HostName:         hostName,
				RenderedHost:     renderedHost,
				Attempts:         1,
				LastError:        err,
				LastAttemptTime:  time.Now(),
			})
		}
		// Also queue bootc upgrade for retry
		*retryQueue = append(*retryQueue, RetryItem{
			ExporterInstance: nil,
			HostName:         hostName,
			RenderedHost:     renderedHost,
			Attempts:         1,
			LastError:        err,
			LastAttemptTime:  time.Now(),
		})
		return
	}

	defer func() {
		_ = hostSsh.Close()
	}()

	// Process exporter instances
	for _, exporterInstance := range exporterInstances {
		if err := e.processExporterInstance(exporterInstance, hostSsh, renderedHost); err != nil {
			fmt.Printf("    ❌ Failed to process %s: %v\n", exporterInstance.Name, err)
			*retryQueue = append(*retryQueue, RetryItem{
				ExporterInstance: exporterInstance,
				HostName:         hostName,
				RenderedHost:     renderedHost,
				Attempts:         1,
				LastError:        err,
				LastAttemptTime:  time.Now(),
			})
		}
	}

	// Handle bootc upgrade
	if err := hostSsh.HandleBootcUpgrade(e.dryRun); err != nil {
		fmt.Printf("    ⚠️  Bootc upgrade error: %v\n", err)
		*retryQueue = append(*retryQueue, RetryItem{
			ExporterInstance: nil,
			HostName:         hostName,
			RenderedHost:     renderedHost,
			Attempts:         1,
			LastError:        err,
			LastAttemptTime:  time.Now(),
		})
	}
}

// processGlobalRetryQueue processes the global retry queue with exponential backoff
func (e *ExporterHostSyncer) processGlobalRetryQueue(retryQueue []RetryItem) error {
	var finalErrors []string

	for len(retryQueue) > 0 {
		var nextRetryQueue []RetryItem
		var itemsToRetry []RetryItem

		// First pass: separate items that are ready to retry from those that need to wait
		for _, retryItem := range retryQueue {
			// Check if we've exceeded max attempts
			if retryItem.Attempts >= e.retryConfig.MaxAttempts {
				if retryItem.ExporterInstance == nil {
					fmt.Printf("💀 Max retry attempts exceeded for bootc upgrade on %s, giving up: %v\n",
						retryItem.HostName, retryItem.LastError)
					finalErrors = append(finalErrors, fmt.Sprintf("bootc upgrade on %s: %v", retryItem.HostName, retryItem.LastError))
				} else {
					fmt.Printf("💀 Max retry attempts exceeded for %s on %s, giving up: %v\n",
						retryItem.ExporterInstance.Name, retryItem.HostName, retryItem.LastError)
					finalErrors = append(finalErrors, fmt.Sprintf("%s on %s: %v", retryItem.ExporterInstance.Name, retryItem.HostName, retryItem.LastError))
				}
				continue
			}

			// Calculate delay since last attempt
			timeSinceLastAttempt := time.Since(retryItem.LastAttemptTime)
			requiredDelay := e.calculateBackoffDelay(retryItem.Attempts - 1)

			// If not enough time has passed, add to next retry queue
			if timeSinceLastAttempt < requiredDelay {
				nextRetryQueue = append(nextRetryQueue, retryItem)
			} else {
				// Ready to retry
				itemsToRetry = append(itemsToRetry, retryItem)
			}
		}

		// Second pass: retry items that are ready
		for _, retryItem := range itemsToRetry {
			fmt.Printf("🔄 Retrying %s on %s (attempt %d/%d)...\n",
				getRetryItemDescription(retryItem), retryItem.HostName, retryItem.Attempts+1, e.retryConfig.MaxAttempts)

			// Create a fresh SSH connection
			hostSsh, err := ssh.NewSSHHostManager(retryItem.RenderedHost)
			if err != nil {
				fmt.Printf("❌ SSH connection failed for %s: %v\n", retryItem.HostName, err)
				e.addToRetryQueue(&retryItem, err, &nextRetryQueue)
				continue
			}

			defer func() {
				_ = hostSsh.Close()
			}()

			// Test the connection
			status, err := hostSsh.Status()
			if err != nil {
				fmt.Printf("❌ SSH connection test failed for %s: %v\n", retryItem.HostName, err)
				e.addToRetryQueue(&retryItem, err, &nextRetryQueue)
				continue
			}

			fmt.Printf("✅ SSH connection established for %s: %s\n", retryItem.HostName, status)

			// Now perform the actual retry operation
			if retryItem.ExporterInstance == nil {
				// This was a bootc upgrade failure
				if err := hostSsh.HandleBootcUpgrade(e.dryRun); err != nil {
					fmt.Printf("❌ Retry failed for bootc upgrade on %s: %v\n", retryItem.HostName, err)
					e.addToRetryQueue(&retryItem, err, &nextRetryQueue)
				} else {
					fmt.Printf("✅ Retry succeeded for bootc upgrade on %s\n", retryItem.HostName)
				}
			} else {
				// This was an exporter instance failure
				if err := e.processExporterInstance(retryItem.ExporterInstance, hostSsh, retryItem.RenderedHost); err != nil {
					fmt.Printf("❌ Retry failed for %s on %s: %v\n", retryItem.ExporterInstance.Name, retryItem.HostName, err)
					e.addToRetryQueue(&retryItem, err, &nextRetryQueue)
				} else {
					fmt.Printf("✅ Retry succeeded for %s on %s\n", retryItem.ExporterInstance.Name, retryItem.HostName)
				}
			}
		}

		retryQueue = nextRetryQueue

		// If there are still items to retry, wait before the next iteration
		if len(retryQueue) > 0 {
			// Find the minimum delay needed for the next retry
			minDelay := e.retryConfig.MaxDelay
			for _, item := range retryQueue {
				requiredDelay := e.calculateBackoffDelay(item.Attempts - 1)
				timeSinceLastAttempt := time.Since(item.LastAttemptTime)
				remainingDelay := requiredDelay - timeSinceLastAttempt
				if remainingDelay < minDelay {
					minDelay = remainingDelay
				}
			}

			if minDelay > 0 {
				// Round to nearest second to avoid decimal display
				roundedDelay := time.Duration(int64(minDelay.Seconds())) * time.Second
				if roundedDelay == 0 && minDelay > 0 {
					roundedDelay = 1 * time.Second
				}
				fmt.Printf("⏳ Waiting %v before next retry cycle...\n", roundedDelay)
				time.Sleep(roundedDelay + 1*time.Second)
			}
		}
	}

	// Return error if any instances failed after all retries
	if len(finalErrors) > 0 {
		return fmt.Errorf("failed to process exporter instances after retries: %s", strings.Join(finalErrors, "; "))
	}

	return nil
}

// SyncExporterHosts synchronizes exporter hosts via SSH
func (e *ExporterHostSyncer) SyncExporterHosts() error {
	fmt.Print("\n🔄 Syncing exporter hosts via SSH ===========================\n")

	// Global retry queue to collect all failed instances
	retryQueue := make([]RetryItem, 0)

	// First pass: process all hosts and collect failures
	for _, host := range e.cfg.Loaded.ExporterHosts {
		exporterInstances := e.cfg.Loaded.GetExporterInstancesByExporterHost(host.Name)
		exporterInstances = e.filterExporterInstances(host.Name, exporterInstances)

		// Skip the host if no viable exporter instances remain
		if len(exporterInstances) == 0 {
			continue
		}

		// Apply templates to the host
		hostCopy := host.DeepCopy()
		if err := e.tapplier.Apply(hostCopy); err != nil {
			return fmt.Errorf("error applying template for %s: %w", host.Name, err)
		}
		// if there are no addresses, skip the host
		if len(hostCopy.Spec.Addresses) == 0 {
			fmt.Printf("    ❌ Skipping %s - no addresses\n", host.Name)
			continue
		}
		fmt.Printf("\n💻  Exporter host: %s\n", hostCopy.Spec.Addresses[0])

		// Process each exporter instance and add failures to global retry queue
		e.processExporterInstancesAndBootc(exporterInstances, host.Name, hostCopy, &retryQueue)
	}

	// Second pass: retry all failed instances globally
	if len(retryQueue) > 0 {
		fmt.Printf("\n🔄 Processing retry queue (%d failed instances) ===========================\n", len(retryQueue))
		if err := e.processGlobalRetryQueue(retryQueue); err != nil {
			return fmt.Errorf("error syncing exporter hosts: %w", err)
		}
	}

	return nil
}
