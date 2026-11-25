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
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

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
	parallelism          int
	// flightctlConfigPath is the flightctl client config to use; empty means where the flightctl CLI keeps it.
	flightctlConfigPath string
}

func NewExporterHostSyncer(cfg *config.Config,
	tapplier *templating.TemplateApplier,
	serviceParametersMap map[string]template.ServiceParameters,
	dryRun, debugConfigs bool,
	exporterFilter *regexp.Regexp,
	parallelism int,
	flightctlConfigPath string) *ExporterHostSyncer {
	return &ExporterHostSyncer{
		cfg:                  cfg,
		tapplier:             tapplier,
		serviceParametersMap: serviceParametersMap,
		dryRun:               dryRun,
		debugConfigs:         debugConfigs,
		exporterFilter:       exporterFilter,
		parallelism:          parallelism,
		flightctlConfigPath:  flightctlConfigPath,
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
	return instance.IsDead()
}

func isExporterInstanceUnmanaged(instance *api.ExporterInstance) (bool, string) {
	return instance.IsUnmanaged()
}

func deadReason(annotation string) string {
	if annotation == "" {
		return "dead"
	}
	return fmt.Sprintf("dead: %s", annotation)
}

func unmanagedReason(annotation string) string {
	if annotation == "" {
		return "missing discovery date"
	}
	return annotation
}

// filterExporterInstances filters instances, checks if all are dead, and handles skip printing
func (e *ExporterHostSyncer) filterExporterInstances(hostName string, exporterInstances []*api.ExporterInstance, out *OutputBuffer) []*api.ExporterInstance {
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

	out.instanceCount = len(exporterInstances)

	// no instances match the filter
	if len(exporterInstances) == 0 {
		return nil
	}

	activeInstances := make([]*api.ExporterInstance, 0, len(exporterInstances))
	inactiveReasons := make([]string, 0, len(exporterInstances))

	for _, exporterInstance := range exporterInstances {
		if exporterInstance == nil {
			continue
		}

		if isDead, deadAnnotation := isExporterInstanceDead(exporterInstance); isDead {
			reason := deadReason(deadAnnotation)
			out.Printf("    📟 Exporter instance: %s skipped - %s\n", exporterInstance.Name, reason)
			inactiveReasons = append(inactiveReasons, fmt.Sprintf("%s %s", exporterInstance.Name, reason))
			continue
		}

		if isUnmanaged, unmanagedAnnotation := isExporterInstanceUnmanaged(exporterInstance); isUnmanaged {
			reason := unmanagedReason(unmanagedAnnotation)
			out.Printf("    📟 Exporter instance: %s skipped - unmanaged: %s\n", exporterInstance.Name, reason)
			inactiveReasons = append(inactiveReasons, fmt.Sprintf("%s unmanaged: %s", exporterInstance.Name, reason))
			continue
		}

		activeInstances = append(activeInstances, exporterInstance)
	}

	out.SetSkippedInstances(inactiveReasons)

	// all instances are dead/unmanaged
	if len(activeInstances) == 0 && len(inactiveReasons) > 0 {
		out.Printf("\n💻  Exporter host: %s skipped - all instances inactive: [%s]\n", hostName, strings.Join(inactiveReasons, ", "))
		out.MarkSkipped(strings.Join(inactiveReasons, ", "))
		return nil
	}

	return activeInstances
}

// processExporterInstance processes a single exporter instance
func (e *ExporterHostSyncer) processExporterInstance(exporterInstance *api.ExporterInstance, hostSsh ssh.HostManager, out *OutputBuffer) error {
	tcfg, err := e.renderExporterInstance(exporterInstance, out)
	if err != nil {
		return err
	}
	return hostSsh.Apply(tcfg, e.dryRun)
}

// renderExporterInstance renders the templates of a single exporter instance
func (e *ExporterHostSyncer) renderExporterInstance(exporterInstance *api.ExporterInstance, out *OutputBuffer) (*api.ExporterConfigTemplate, error) {
	// filterExporterInstances only passes active (non-dead, managed) instances here.

	out.Printf("    📟 Exporter instance: %s\n", exporterInstance.Name)
	errName := "ExporterInstance:" + exporterInstance.Name

	et, err := template.NewExporterInstanceTemplater(e.cfg, exporterInstance)
	if err != nil {
		return nil, fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
	}

	spRef := exporterInstance.Spec.JumpstarterInstanceRef.Name + ":" + exporterInstance.Name
	serviceParameters, ok := e.serviceParametersMap[spRef]
	if !ok {
		return nil, fmt.Errorf("service parameters not found for %s", spRef)
	}
	et.SetServiceParameters(serviceParameters)

	_, err = et.RenderTemplateLabels()
	if err != nil {
		return nil, fmt.Errorf("error creating ExporterInstanceTemplater for %s : %w", errName, err)
	}

	tcfg, err := et.RenderTemplateConfig()
	if err != nil {
		return nil, fmt.Errorf("error rendering template config for %s : %w", errName, err)
	}

	if e.debugConfigs {
		out.Printf("--- 📄 Config Template %s\n", strings.Repeat("─", 40))
		out.Printf("%s\n", tcfg.Spec.ConfigTemplate)
		if tcfg.Spec.SystemdContainerTemplate != "" {
			out.Printf("  - ⚙️  Systemd Container Template %s\n", strings.Repeat("─", 30))
			out.Printf("%s\n", tcfg.Spec.SystemdContainerTemplate)
		}
		if tcfg.Spec.SystemdServiceTemplate != "" {
			out.Printf("  - 🔧 Systemd Service Template %s\n", strings.Repeat("─", 31))
			out.Printf("%s\n", tcfg.Spec.SystemdServiceTemplate)
		}
		out.Printf("%s\n", strings.Repeat("─", 60))
	}

	return tcfg, nil
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

// processExporterInstancesAndBootc processes exporter instances and collects failures on the OutputBuffer
func (e *ExporterHostSyncer) processExporterInstancesAndBootc(exporterInstances []*api.ExporterInstance, hostName string, renderedHost *api.ExporterHost, out *OutputBuffer) {
	// Create SSH connection
	hostSsh, err := ssh.NewSSHHostManager(renderedHost)
	if err == nil {
		// Wire the SSH host manager to write to our buffer
		hostSsh.SetWriter(out.Writer())
		_, err = hostSsh.Status()
	}
	if err != nil {
		out.Printf("    ❌ Failed to create/test SSH connection: %v\n", err)
		out.MarkError()
		// Queue all exporter instances for retry
		for _, exporterInstance := range exporterInstances {
			out.AddRetryItem(RetryItem{
				ExporterInstance: exporterInstance,
				HostName:         hostName,
				RenderedHost:     renderedHost,
				Attempts:         1,
				LastError:        err,
				LastAttemptTime:  time.Now(),
			})
		}
		// Also queue bootc upgrade for retry
		out.AddRetryItem(RetryItem{
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
		if err := e.processExporterInstance(exporterInstance, hostSsh, out); err != nil {
			out.Printf("    ❌ Failed to process %s: %v\n", exporterInstance.Name, err)
			out.MarkError()
			out.AddRetryItem(RetryItem{
				ExporterInstance: exporterInstance,
				HostName:         hostName,
				RenderedHost:     renderedHost,
				Attempts:         1,
				LastError:        err,
				LastAttemptTime:  time.Now(),
			})
		} else {
			out.MarkChanged()
		}
	}

	// Handle bootc upgrade
	if err := hostSsh.HandleBootcUpgrade(e.dryRun); err != nil {
		out.Printf("    ⚠️  Bootc upgrade error: %v\n", err)
		out.MarkError()
		out.AddRetryItem(RetryItem{
			ExporterInstance: nil,
			HostName:         hostName,
			RenderedHost:     renderedHost,
			Attempts:         1,
			LastError:        err,
			LastAttemptTime:  time.Now(),
		})
	}
}

// hostWork represents a unit of work for processing a single host
type hostWork struct {
	host             *api.ExporterHost
	hostName         string
	instances        []*api.ExporterInstance
	totalInstances   int
	skippedInstances []string
	skippedOutput    string
}

// newOutput starts the output of a host under label, with the messages about its skipped instances.
func (w hostWork) newOutput(label string) *OutputBuffer {
	out := NewOutputBuffer(label, w.totalInstances)
	out.SetSkippedInstances(w.skippedInstances)
	out.Printf("\n💻  Exporter host: %s\n", label)
	out.Printf("%s", w.skippedOutput)
	return out
}

// processGlobalRetryQueue processes the global retry queue with exponential backoff
// Retries are parallelized by host, using the same concurrency limit as the initial pass.
// Returns the number of items that succeeded on retry.
func (e *ExporterHostSyncer) processGlobalRetryQueue(retryQueue []RetryItem, printer *SyncPrinter) (int32, error) {
	var finalErrors []string
	var totalSucceeded int32

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

		// Group ready items by host for parallel retry
		hostRetryGroups := make(map[string][]RetryItem)
		for _, item := range itemsToRetry {
			hostRetryGroups[item.HostName] = append(hostRetryGroups[item.HostName], item)
		}

		// Process host groups in parallel
		var retryMu sync.Mutex
		g, _ := errgroup.WithContext(context.Background())
		if e.parallelism > 0 {
			g.SetLimit(e.parallelism)
		}

		var roundSucceeded atomic.Int32
		for hostName, items := range hostRetryGroups {
			g.Go(func() error {
				out := NewOutputBuffer(hostName, len(items))
				out.Printf("\n🔄 Retrying %d items on %s...\n", len(items), hostName)

				localRetries, succeeded := e.processRetryGroup(items, out)
				out.Done()
				roundSucceeded.Add(int32(succeeded))

				retryMu.Lock()
				nextRetryQueue = append(nextRetryQueue, localRetries...)
				retryMu.Unlock()

				printer.FlushBuffer(out)
				return nil
			})
		}
		_ = g.Wait()
		totalSucceeded += roundSucceeded.Load()

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
		return totalSucceeded, fmt.Errorf("failed to process exporter instances after retries: %s", strings.Join(finalErrors, "; "))
	}

	return totalSucceeded, nil
}

// processRetryGroup retries a group of items for a single host (sequential within the host).
// Returns the remaining retry items and the number of items that succeeded.
func (e *ExporterHostSyncer) processRetryGroup(items []RetryItem, out *OutputBuffer) ([]RetryItem, int) {
	var localRetries []RetryItem
	succeeded := 0

	// Create a single SSH connection for all items on this host
	var hostSsh ssh.HostManager
	var sshErr error

	if len(items) > 0 {
		hostSsh, sshErr = ssh.NewSSHHostManager(items[0].RenderedHost)
		if sshErr == nil {
			hostSsh.SetWriter(out.Writer())
			_, sshErr = hostSsh.Status()
		}
	}

	if sshErr != nil {
		out.Printf("❌ SSH connection failed for %s: %v\n", items[0].HostName, sshErr)
		out.MarkError()
		for i := range items {
			e.addToRetryQueue(&items[i], sshErr, &localRetries)
		}
		return localRetries, 0
	}

	if hostSsh != nil {
		defer func() {
			_ = hostSsh.Close()
		}()
	}

	for i := range items {
		retryItem := &items[i]
		out.Printf("  🔄 Retrying %s (attempt %d/%d)...\n",
			getRetryItemDescription(*retryItem), retryItem.Attempts+1, e.retryConfig.MaxAttempts)

		if retryItem.ExporterInstance == nil {
			// This was a bootc upgrade failure
			if err := hostSsh.HandleBootcUpgrade(e.dryRun); err != nil {
				out.Printf("  ❌ Retry failed for bootc upgrade: %v\n", err)
				out.MarkError()
				e.addToRetryQueue(retryItem, err, &localRetries)
			} else {
				out.Printf("  ✅ Retry succeeded for bootc upgrade\n")
				out.MarkChanged()
				succeeded++
			}
		} else {
			// This was an exporter instance failure
			if err := e.processExporterInstance(retryItem.ExporterInstance, hostSsh, out); err != nil {
				out.Printf("  ❌ Retry failed for %s: %v\n", retryItem.ExporterInstance.Name, err)
				out.MarkError()
				e.addToRetryQueue(retryItem, err, &localRetries)
			} else {
				out.Printf("  ✅ Retry succeeded for %s\n", retryItem.ExporterInstance.Name)
				out.MarkChanged()
				succeeded++
			}
		}
	}

	return localRetries, succeeded
}

// SyncExporterHosts synchronizes exporter hosts via SSH or flightctl, processing hosts in parallel.
func (e *ExporterHostSyncer) SyncExporterHosts() error {
	fmt.Print("\n🔄 Syncing exporter hosts ===========================\n")

	printer := NewSyncPrinter()

	// Pre-filter and template all hosts (fast, no SSH involved)
	var sshWork, flightctlWork []hostWork
	for _, host := range e.cfg.Loaded.ExporterHosts {
		exporterInstances := e.cfg.Loaded.GetExporterInstancesByExporterHost(host.Name)

		// Use a temporary buffer for filtering output (e.g. "all dead" messages)
		filterOut := NewOutputBuffer(host.Name, len(exporterInstances))
		filteredInstances := e.filterExporterInstances(host.Name, exporterInstances, filterOut)

		// Skip the host if no viable exporter instances remain
		if len(filteredInstances) == 0 {
			// Flush any filter messages (like "all dead" skips)
			if filterOut.skipped || filterOut.buf.Len() > 0 {
				printer.FlushBuffer(filterOut)
			}
			continue
		}

		// Apply templates to the host
		hostCopy := host.DeepCopy()
		if err := e.tapplier.Apply(hostCopy); err != nil {
			return fmt.Errorf("error applying template for %s: %w", host.Name, err)
		}

		w := hostWork{
			host:             hostCopy,
			hostName:         host.Name,
			instances:        filteredInstances,
			totalInstances:   filterOut.instanceCount,
			skippedInstances: filterOut.skippedInstances,
			skippedOutput:    filterOut.buf.String(),
		}
		// Lint rejects a host with both management.ssh and management.flightctl
		switch {
		case hostCopy.Spec.Management.FlightCtl != nil:
			flightctlWork = append(flightctlWork, w)
		case len(hostCopy.Spec.Addresses) == 0:
			fmt.Printf("    ❌ Skipping %s - no addresses\n", host.Name)
		default:
			sshWork = append(sshWork, w)
		}
	}

	// Only a run that reaches a flightctl host reads the flightctl login. A login that cannot be read
	// fails the flightctl hosts like any other flightctl error, and the SSH hosts still sync.
	flightctlClient := sync.OnceValues(e.newFlightctlClient)

	// Process hosts in parallel
	var resultsMu sync.Mutex
	retryQueue := make([]RetryItem, 0)
	var flightctlErrs []error

	g, _ := errgroup.WithContext(context.Background())
	if e.parallelism > 0 {
		g.SetLimit(e.parallelism)
	}

	for _, w := range sshWork {
		g.Go(func() error {
			out := w.newOutput(w.host.Spec.Addresses[0])
			e.processExporterInstancesAndBootc(w.instances, w.hostName, w.host, out)
			out.Done()

			// Collect retry items under lock
			resultsMu.Lock()
			retryQueue = append(retryQueue, out.retryItems...)
			resultsMu.Unlock()

			// Atomically flush output for this host
			printer.FlushBuffer(out)
			return nil
		})
	}
	// A flightctl update is one API call per host, so it is not retried: a failure is final for this run
	for _, w := range flightctlWork {
		g.Go(func() error {
			out := w.newOutput(w.hostName + " (flightctl)")
			if err := e.syncFlightctlHost(flightctlClient, w, out); err != nil {
				out.Printf("    ❌ Failed to sync via flightctl: %v\n", err)
				out.MarkError()
				resultsMu.Lock()
				flightctlErrs = append(flightctlErrs, fmt.Errorf("flightctl host %s: %w", w.hostName, err))
				resultsMu.Unlock()
			}
			out.Done()
			printer.FlushBuffer(out)
			return nil
		})
	}
	_ = g.Wait()

	slices.SortFunc(flightctlErrs, func(a, b error) int { return strings.Compare(a.Error(), b.Error()) })
	flightctlErr := errors.Join(flightctlErrs...)

	// Second pass: retry all failed instances globally
	if len(retryQueue) > 0 {
		retryTotal := int32(len(retryQueue))
		fmt.Printf("\n🔄 Processing retry queue (%d failed items) ===========================\n", retryTotal)
		retrySucceeded, err := e.processGlobalRetryQueue(retryQueue, printer)
		printer.AddRetryStats(retryTotal, retrySucceeded)
		if err != nil {
			return errors.Join(fmt.Errorf("error syncing exporter hosts: %w", err), flightctlErr)
		}
	}

	// Print final summary
	printer.PrintSummary()

	return flightctlErr
}
