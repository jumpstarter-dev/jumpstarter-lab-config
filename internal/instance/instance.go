package instance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managedByAnnotation = "jumpstarter-lab-config"
)

// Instance wraps a Kubernetes client and provides methods for operating on Jumpstarter resources
type Instance struct {
	client client.Client
	config *v1alphaConfig.JumpstarterInstance
	dryRun bool
	prune  bool
}

// NewInstance creates a new Instance from a JumpstarterInstance and optional kubeconfig string
// If kubeconfigStr is empty, it will try to load from environment/standard kubeconfig file
// This function ensures proper scheme registration for all custom API types
func NewInstance(instance *v1alphaConfig.JumpstarterInstance, kubeconfigStr string, dryRun, prune bool) (*Instance, error) {
	// Validate the instance
	if err := validateInstance(instance); err != nil {
		return nil, fmt.Errorf("invalid instance: %w", err)
	}

	// Get the context from the instance
	contextName := instance.Spec.KubeContext

	// Create a custom scheme with our API types
	customScheme := scheme.Scheme

	// Add jumpstarter-controller API types (which includes ExporterList)
	if err := v1alpha1.AddToScheme(customScheme); err != nil {
		return nil, fmt.Errorf("failed to add jumpstarter-controller API types to scheme: %w", err)
	}

	// Also add local API types if needed
	if err := v1alphaConfig.AddToScheme(customScheme); err != nil {
		return nil, fmt.Errorf("failed to add local API types to scheme: %w", err)
	}

	// Create a kube client
	kc := NewKubeClient()

	var restConfig *rest.Config
	var err error

	// If kubeconfig string is provided, use it
	if kubeconfigStr != "" {
		restConfig, err = kc.getConfigWithContext([]byte(kubeconfigStr), contextName)
		if err != nil {
			return nil, fmt.Errorf("failed to get config with context %s: %w", contextName, err)
		}
	} else {
		// Otherwise, try to load from environment/standard kubeconfig file
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
		restConfig, err = kc.buildConfigFromFileWithContext(kubeconfigPath, contextName)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// Create the client with our custom scheme
	c, err := client.New(restConfig, client.Options{Scheme: customScheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Instance{
		client: c,
		config: instance,
		dryRun: dryRun,
		prune:  prune,
	}, nil
}

// GetClient returns the underlying Kubernetes client
func (i *Instance) GetClient() client.Client {
	return i.client
}

// GetConfig returns the JumpstarterInstance configuration
func (i *Instance) GetConfig() *v1alphaConfig.JumpstarterInstance {
	return i.config
}

// ListExporters lists all exporters in the instance's namespace
func (i *Instance) ListExporters(ctx context.Context) (*v1alpha1.ExporterList, error) {
	return i.listExporters(ctx)
}

// ListClients lists all clients in the instance's namespace
func (i *Instance) ListClients(ctx context.Context) (*v1alpha1.ClientList, error) {
	return i.listClients(ctx)
}

// GetClientByName retrieves a specific client by name
func (i *Instance) GetClientByName(ctx context.Context, name string) (*v1alpha1.Client, error) {
	return i.getClientByName(ctx, name)
}

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

// listClients lists all clients in the instance's namespace
func (i *Instance) listClients(ctx context.Context) (*v1alpha1.ClientList, error) {
	clients := &v1alpha1.ClientList{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		// If no namespace specified, list from all namespaces
		err := i.client.List(ctx, clients)
		return clients, err
	}

	err := i.client.List(ctx, clients, client.InNamespace(namespace))
	return clients, err
}

// getClientByName retrieves a specific client by name
func (i *Instance) getClientByName(ctx context.Context, name string) (*v1alpha1.Client, error) {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required to get client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	return clientObj, err
}

// updateClient updates a client
func (i *Instance) updateClient(ctx context.Context, oldClientObj, clientObj *v1alpha1.Client) error {
	// Create a copy of the old object to preserve ResourceVersion and other metadata
	updatedClient := oldClientObj.DeepCopy()

	// Update the spec and other fields from the new config
	updatedClient.Spec = clientObj.Spec
	updatedClient.Labels = clientObj.Labels

	// Prepare metadata (annotations, namespace, etc.)
	// For updates, we want to preserve existing annotations and merge new ones
	i.prepareMetadata(&updatedClient.ObjectMeta, clientObj.Annotations)
	i.printDiff(oldClientObj, updatedClient, "client", updatedClient.Name)
	if i.dryRun {
		return nil
	}

	return i.client.Update(ctx, updatedClient)
}

// createClient creates a new client
func (i *Instance) createClient(ctx context.Context, clientObj *v1alpha1.Client) error {
	// Prepare metadata (annotations, namespace, etc.)
	i.prepareMetadata(&clientObj.ObjectMeta, clientObj.Annotations)

	if i.dryRun {
		fmt.Printf("➕ [%s] dry run: Would create client %s in namespace %s\n", i.config.Name, clientObj.Name, clientObj.Namespace)
		return nil
	}

	return i.client.Create(ctx, clientObj)
}

// deleteClient deletes a client by name
func (i *Instance) deleteClient(ctx context.Context, name string) error {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("namespace is required to delete client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	if err != nil {
		return fmt.Errorf("failed to get client %s: %w", name, err)
	}

	if i.dryRun {
		fmt.Printf("🗑️ [%s] dry run: Would delete client %s in namespace %s\n", i.config.Name, name, namespace)
		return nil
	}

	return i.client.Delete(ctx, clientObj)
}

func (i *Instance) prepareMetadata(metadata *metav1.ObjectMeta, newAnnotations map[string]string) {

	// Initialize annotations if nil
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}

	// Merge new annotations into existing ones
	for key, value := range newAnnotations {
		metadata.Annotations[key] = value
	}

	// Ensure the managed-by annotation is set
	metadata.Annotations["managed-by"] = managedByAnnotation

	// Set namespace if not already set
	if metadata.Namespace == "" {
		metadata.Namespace = i.config.Spec.Namespace
	}
}

// validateInstance performs basic validation on a JumpstarterInstance
func validateInstance(instance *v1alphaConfig.JumpstarterInstance) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	// Additional validation can be added here as needed
	// For example, checking if required fields are present

	return nil
}

// printDiff prints a diff between two objects, ignoring Kubernetes metadata fields
func (i *Instance) printDiff(oldObj, newObj interface{}, objType, objName string) {
	// Options to ignore Kubernetes metadata fields that change frequently
	ignoreOpts := []cmp.Option{
		cmpopts.IgnoreFields(metav1.ObjectMeta{}, "Generation", "CreationTimestamp", "ResourceVersion", "UID", "ManagedFields"),
		cmpopts.IgnoreFields(v1alpha1.Exporter{}, "Status"),
		cmpopts.IgnoreFields(v1alpha1.Client{}, "Status"),
	}

	diff := cmp.Diff(oldObj, newObj, ignoreOpts...)
	if diff != "" {
		fmt.Printf("📝 [%s] dry run: Would update %s %s, diff: %s\n", i.config.Name, objType, objName, diff)
	} else {
		fmt.Printf("✅ [%s] dry run: No changes needed for %s %s\n", i.config.Name, objType, objName)
	}
}

func (i *Instance) SyncClients(ctx context.Context, cfg *config.Config) error {
	fmt.Printf("🔄 [%s] Syncing clients ===========================\n", i.config.Name)
	instanceClients, err := i.listClients(ctx)
	if err != nil {
		return fmt.Errorf("[%s] failed to list clients: %w", i.config.Name, err)
	}

	configClientMap := cfg.Loaded.Clients

	// create a clientMap from instanceClients
	instanceClientMap := make(map[string]v1alpha1.Client)
	for _, instClient := range instanceClients.Items {
		instanceClientMap[instClient.Name] = instClient
	}

	// delete clients that are not in config
	for _, instanceClient := range instanceClients.Items {
		if _, ok := configClientMap[instanceClient.Name]; !ok {
			err := i.deleteClient(ctx, instanceClient.Name)
			if err != nil {
				return fmt.Errorf("[%s] failed to delete client %s: %w", i.config.Name, instanceClient.Name, err)
			}
		}
	}

	// create clients that are in config but not in instance
	for _, cfgClient := range configClientMap {
		if _, ok := instanceClientMap[cfgClient.Name]; !ok {
			err := i.createClient(ctx, cfgClient)
			if err != nil {
				return fmt.Errorf("[%s] failed to create client %s: %w", i.config.Name, cfgClient.Name, err)
			}
		}
	}

	// update clients that are in both config and instance
	for _, instanceClient := range instanceClients.Items {
		if cfgClient, ok := configClientMap[instanceClient.Name]; ok {
			err := i.updateClient(ctx, &instanceClient, cfgClient)
			if err != nil {
				return fmt.Errorf("[%s] failed to update client %s: %w", i.config.Name, instanceClient.Name, err)
			}
		}
	}

	return nil
}
