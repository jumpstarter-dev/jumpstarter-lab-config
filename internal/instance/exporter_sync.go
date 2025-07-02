package instance

import (
	"context"
	"fmt"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
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

// updateExporter updates an exporter
//
//nolint:unused
func (i *Instance) updateExporter(ctx context.Context, oldExporter, exporter *v1alpha1.Exporter) error {
	// Create a copy of the current object to preserve ResourceVersion and other metadata
	updatedExporter := oldExporter.DeepCopy()

	// Update the spec and other fields from the new config
	updatedExporter.Spec = exporter.Spec
	updatedExporter.Labels = exporter.Labels

	// Prepare metadata (annotations, namespace, etc.)
	// For updates, we want to preserve existing annotations and merge new ones
	i.prepareMetadata(&updatedExporter.ObjectMeta, exporter.Annotations)
	i.printDiff(oldExporter, updatedExporter, "exporter", updatedExporter.Name)
	if i.dryRun {
		return nil
	}

	return i.client.Update(ctx, updatedExporter)
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

func (i *Instance) SyncExporters(ctx context.Context, cfg *config.Config) error {
	fmt.Printf("🔄 [%s] Syncing exporters ===========================\n", i.config.Name)
	instanceExporters, err := i.listExporters(ctx)
	if err != nil {
		return fmt.Errorf("[%s] failed to list exporters: %w", i.config.Name, err)
	}

	// convert configExporterMap to a map of exporter name to exporter objects
	configExporterMap := make(map[string]v1alpha1.Exporter)
	for _, cfgExporterInstance := range cfg.Loaded.ExporterInstances {
		exporterObj := cfgExporterInstance.GetExporterObjectForInstance(i.config.Name)
		if exporterObj != nil {
			configExporterMap[cfgExporterInstance.Name] = *exporterObj
		}
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
				return fmt.Errorf("[%s] failed to delete exporter %s: %w", i.config.Name, instanceExporter.Name, err)
			}
		}
	}

	// create exporters that are in config but not in instance
	for _, cfgExporter := range configExporterMap {
		// Use the helper method to get the Exporter object for this instance

		if _, ok := instanceExporterMap[cfgExporter.Name]; !ok {
			err := i.createExporter(ctx, &cfgExporter)
			if err != nil {
				return fmt.Errorf("[%s] failed to create exporter %s: %w", i.config.Name, cfgExporter.Name, err)
			}
		}
	}

	// update exporters that are in both config and instance
	for _, instanceExporter := range instanceExporters.Items {
		if exporterObj, ok := configExporterMap[instanceExporter.Name]; ok {

			err := i.updateExporter(ctx, &instanceExporter, &exporterObj)
			if err != nil {
				return fmt.Errorf("[%s] failed to update exporter %s: %w", i.config.Name, instanceExporter.Name, err)
			}
		}
	}

	return nil
}
