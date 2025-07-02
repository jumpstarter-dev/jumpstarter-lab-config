package instance

import (
	"context"
	"fmt"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
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

// getExporter retrieves a specific exporter by name
func (i *Instance) getExporter(ctx context.Context, name string) (*v1alpha1.Exporter, error) {
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
	// Get the current version from the cluster to preserve ResourceVersion

	// Create a copy of the current object to preserve ResourceVersion and other metadata
	updatedExporter := oldExporter.DeepCopy()

	// Update the spec and other fields from the new config
	updatedExporter.Spec = exporter.Spec
	updatedExporter.Labels = exporter.Labels
	// Prepare metadata (annotations, namespace, etc.)
	i.prepareMetadata(&updatedExporter.ObjectMeta, exporter.Annotations)
	i.printDiff(oldExporter.TypeMeta, updatedExporter, "exporter", updatedExporter.Name)
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

	if i.dryRun {
		fmt.Printf("🗑️ [%s] dry run: Would delete exporter %s in namespace %s\n", i.config.Name, name, namespace)
		return nil
	}

	return i.client.Delete(ctx, exporter)
}
