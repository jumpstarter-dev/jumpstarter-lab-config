package instance

import (
	"context"
	"fmt"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	v1alpha1Config "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// listExporters lists all exporters in the instance's namespace
func (i *Instance) listExporters(ctx context.Context) (*v1alpha1.ExporterList, error) {
	exporters := &v1alpha1.ExporterList{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		// If no namespace specified, list from all namespaces
		err := i.client.List(ctx, exporters)
		return exporters, err
	}

	err := i.client.List(ctx, exporters, client.InNamespace(namespace))
	return exporters, err
}

// getExporterByName retrieves a specific exporter by name
func (i *Instance) getExporterByName(ctx context.Context, name string) (*v1alpha1.Exporter, error) {
	exporter := &v1alpha1.Exporter{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required to get exporter %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, exporter)
	return exporter, err
}

// updateExporter updates an exporter with retry logic to handle conflicts
//
//nolint:unused
func (i *Instance) updateExporter(ctx context.Context, oldExporter, exporter *v1alpha1.Exporter) error {
	maxRetries := 10
	baseDelay := 100 * time.Millisecond

	latestExporter := oldExporter

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Fetch the latest version of the exporter to get current ResourceVersion

		// Create a copy of the latest object to preserve ResourceVersion and other metadata
		updatedExporter := latestExporter.DeepCopy()

		// Update the spec and other fields from the new config
		updatedExporter.Spec = exporter.Spec
		updatedExporter.Labels = exporter.Labels

		// Prepare metadata (annotations, namespace, etc.)
		// For updates, we want to preserve existing annotations and merge new ones
		i.prepareMetadata(&updatedExporter.ObjectMeta, exporter.Annotations)

		// Only print diff on first attempt to avoid spam
		if attempt == 0 {
			changed := i.checkAndPrintDiff(oldExporter, updatedExporter, "exporter", updatedExporter.Name, i.dryRun)
			if !changed {
				return nil
			}
		}

		if i.dryRun {
			return nil
		}

		err := i.client.Update(ctx, updatedExporter)
		if err == nil {
			// Success
			return nil
		}

		// Check if this is a conflict error
		if i.isConflictError(err) {
			if attempt < maxRetries-1 {
				// Calculate delay with exponential backoff and jitter
				delay := time.Duration(attempt+1) * baseDelay
				if delay > 2*time.Second {
					delay = 2 * time.Second
				}
				fmt.Printf("⚠️  [%s] Conflict updating exporter %s, retrying in %v (attempt %d/%d)\n",
					i.config.Name, exporter.Name, delay, attempt+1, maxRetries)
				time.Sleep(delay)
				latestExporter = &v1alpha1.Exporter{}
				err := i.client.Get(ctx, client.ObjectKey{
					Namespace: i.config.Spec.Namespace,
					Name:      exporter.Name,
				}, latestExporter)
				if err != nil {
					return fmt.Errorf("failed to fetch latest exporter %s: %w", exporter.Name, err)
				}
				continue
			}
		}

		// Return the error if it's not a conflict or we've exhausted retries
		return fmt.Errorf("failed to update exporter %s after %d attempts: %w", exporter.Name, attempt+1, err)
	}

	return fmt.Errorf("failed to update exporter %s after %d retries due to conflicts", exporter.Name, maxRetries)
}

// createExporter creates a new exporter
//
//nolint:unused
func (i *Instance) createExporter(ctx context.Context, exporter *v1alpha1.Exporter) error {
	// Prepare metadata (annotations, namespace, etc.)
	i.prepareMetadata(&exporter.ObjectMeta, exporter.Annotations)

	if i.dryRun {
		fmt.Printf("➕ [%s] dry run: Would create exporter %s in namespace %s\n", i.config.Name, exporter.Name, exporter.Namespace)
		return nil
	} else {
		fmt.Printf("➕ [%s] Creating exporter %s in namespace %s\n", i.config.Name, exporter.Name, exporter.Namespace)
	}

	return i.client.Create(ctx, exporter)
}

func (i *Instance) waitExporterCredentials(ctx context.Context, exporter *v1alpha1.Exporter) (*template.ServiceParameters, error) {
	maxRetries := 10
	retryDelay := 1 * time.Second
	var err error

	// the exporter object needs the namespace of the jumpstarter instance
	exporter.Namespace = i.config.Spec.Namespace
	for r := 0; r < maxRetries; r++ {
		var serviceParameters *template.ServiceParameters
		serviceParameters, err = i.getExporterCredentials(ctx, exporter)
		if err != nil {
			fmt.Printf("⌛ [%s] Waiting for exporter credentials for %s in namespace %s\n", i.config.Name, exporter.Name, exporter.Namespace)
		}
		if serviceParameters != nil {
			return serviceParameters, nil
		}
		time.Sleep(retryDelay)
		retryDelay *= 2
		if retryDelay > 10*time.Second {
			retryDelay = 10 * time.Second
		}
	}
	return nil, fmt.Errorf("failed to get exporter credentials after %d retries, last error: %w", maxRetries, err)
}

func (i *Instance) getExporterCredentials(ctx context.Context, exporter *v1alpha1.Exporter) (*template.ServiceParameters, error) {
	exporterObj := &v1alpha1.Exporter{}
	err := i.client.Get(ctx, client.ObjectKey{Namespace: exporter.Namespace, Name: exporter.Name}, exporterObj)
	if err != nil {
		return nil, fmt.Errorf("failed to get exporter %s: %w", exporter.Name, err)
	}

	if exporterObj.Status.Credential == nil {
		return nil, nil
	}

	secret := &corev1.Secret{}
	if err = i.client.Get(ctx, client.ObjectKey{Namespace: exporter.Namespace, Name: exporterObj.Status.Credential.Name}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", exporterObj.Status.Credential.Name, err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return nil, fmt.Errorf("secret %s does not contain a token", exporterObj.Status.Credential.Name)
	}

	return &template.ServiceParameters{
		Token: string(token),
		TlsCA: "", // TODO: add tls ca when we have it
	}, nil
}

// deleteExporter deletes an exporter by name
//
//nolint:unused
func (i *Instance) deleteExporter(ctx context.Context, name string) error {
	exporter := &v1alpha1.Exporter{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("namespace is required to delete exporter %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, exporter)
	if err != nil {
		return fmt.Errorf("failed to get exporter %s: %w", name, err)
	}

	if i.dryRun || i.prune {
		fmt.Printf("🗑️ [%s] dry run / don't prune: Would delete exporter %s in namespace %s\n", i.config.Name, name, namespace)
		return nil
	}

	return i.client.Delete(ctx, exporter)
}

func (i *Instance) SyncExporters(ctx context.Context, cfg *config.Config, filter *regexp.Regexp) (map[string]template.ServiceParameters, error) {
	serviceParametersMap := make(map[string]template.ServiceParameters)
	fmt.Printf("\n🔄 [%s] Syncing exporters ===========================\n\n", i.config.Name)
	instanceExporters, err := i.listExporters(ctx)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to list exporters: %w", i.config.Name, err)
	}

	// convert configExporterMap to a map of exporter name to exporter objects
	configExporterMap := make(map[string]v1alpha1.Exporter)
	for _, cfgExporterInstance := range cfg.Loaded.ExporterInstances {
		exporterObj, err := GetExporterObjectForInstance(cfg, cfgExporterInstance, i.config.Name)
		if err != nil {
			return nil, fmt.Errorf("[%s] failed to get exporter object for instance %s: %w", i.config.Name, cfgExporterInstance.Name, err)
		}
		if exporterObj != nil {
			configExporterMap[cfgExporterInstance.Name] = *exporterObj
		}
	}

	// Apply filter if provided
	if filter != nil {
		filteredInstanceItems := []v1alpha1.Exporter{}
		for _, item := range instanceExporters.Items {
			if filter.MatchString(item.Name) {
				filteredInstanceItems = append(filteredInstanceItems, item)
			}
		}
		instanceExporters.Items = filteredInstanceItems

		filteredConfigExporterMap := make(map[string]v1alpha1.Exporter)
		for name, exporterObj := range configExporterMap {
			if filter.MatchString(name) {
				filteredConfigExporterMap[name] = exporterObj
			}
		}
		configExporterMap = filteredConfigExporterMap
	}

	// create a exporterMap from exporters in the cluster
	instanceExporterMap := make(map[string]v1alpha1.Exporter)
	for _, instExporter := range instanceExporters.Items {
		instanceExporterMap[instExporter.Name] = instExporter
	}

	// delete exporters that are not in config
	for _, instanceExporter := range instanceExporters.Items {
		if _, ok := configExporterMap[instanceExporter.Name]; !ok {
			err := i.deleteExporter(ctx, instanceExporter.Name)
			if err != nil {
				return nil, fmt.Errorf("[%s] failed to delete exporter %s: %w", i.config.Name, instanceExporter.Name, err)
			}
		}
	}

	// create exporters that are in config but not in instance
	for _, cfgExporter := range configExporterMap {
		// Use the helper method to get the Exporter object for this instance

		if _, ok := instanceExporterMap[cfgExporter.Name]; !ok {
			err := i.createExporter(ctx, &cfgExporter)
			if err != nil {
				return nil, fmt.Errorf("[%s] failed to create exporter %s: %w", i.config.Name, cfgExporter.Name, err)
			}
		}
		var serviceParameters *template.ServiceParameters
		if !i.dryRun {
			serviceParameters, err = i.waitExporterCredentials(ctx, &cfgExporter)
			if err != nil {
				return nil, fmt.Errorf("[%s] failed to wait for exporter credentials for %s: %w", i.config.Name, cfgExporter.Name, err)
			}
		} else {
			serviceParameters = &template.ServiceParameters{
				Token: "dry-run",
				TlsCA: "",
			}
		}
		serviceParametersMap[cfgExporter.Name] = *serviceParameters
	}

	// update exporters that are in both config and instance
	for _, instanceExporter := range instanceExporters.Items {
		if exporterObj, ok := configExporterMap[instanceExporter.Name]; ok {

			err := i.updateExporter(ctx, &instanceExporter, &exporterObj)
			if err != nil {
				return nil, fmt.Errorf("[%s] failed to update exporter %s: %w", i.config.Name, instanceExporter.Name, err)
			}

			serviceParameters, err := i.waitExporterCredentials(ctx, &exporterObj)
			if err != nil {
				return nil, fmt.Errorf("[%s] failed to wait for exporter credentials for %s: %w", i.config.Name, exporterObj.Name, err)
			}
			serviceParametersMap[exporterObj.Name] = *serviceParameters
		}
	}

	return serviceParametersMap, nil
}

func GetExporterObjectForInstance(cfg *config.Config, e *v1alpha1Config.ExporterInstance, jumpstarterInstance string) (*v1alpha1.Exporter, error) {
	// If this exporter instance is targeting the given jumpstarter instance, return the exporter object
	if e.Spec.JumpstarterInstanceRef.Name == jumpstarterInstance {
		// by default use the exporter instance metadata
		metadata := e.ObjectMeta.DeepCopy()
		metadata.Labels = e.Spec.Labels

		// but, if the exporter instance has a config template, we need to render
		// the labels based on the underlying template instead
		if e.HasConfigTemplate() {
			et, err := template.NewExporterInstanceTemplater(cfg, e)
			if err != nil {
				return nil, fmt.Errorf("error creating ExporterInstanceTemplater for ExporterInstance %s : %w", e.Name, err)
			}
			metadata = e.ObjectMeta.DeepCopy()
			metadata.Labels, err = et.RenderTemplateLabels()
			if err != nil {
				return nil, fmt.Errorf("error rendering labels for ExporterInstance %s : %w", e.Name, err)
			}
		}
		return &v1alpha1.Exporter{
			TypeMeta:   e.TypeMeta,
			ObjectMeta: *metadata,
			Spec: v1alpha1.ExporterSpec{
				Username: &e.Spec.Username,
			},
		}, nil
	}
	return nil, nil
}

// isConflictError checks if an error is a Kubernetes conflict error
func (i *Instance) isConflictError(err error) bool {
	return apierrors.IsConflict(err)
}
