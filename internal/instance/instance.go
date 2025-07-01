package instance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Instance wraps a Kubernetes client and provides methods for operating on Jumpstarter resources
type Instance struct {
	client client.Client
	config *v1alphaConfig.JumpstarterInstance
}

// NewInstance creates a new Instance from a JumpstarterInstance and optional kubeconfig string
// If kubeconfigStr is empty, it will try to load from environment/standard kubeconfig file
// This function ensures proper scheme registration for all custom API types
func NewInstance(instance *v1alphaConfig.JumpstarterInstance, kubeconfigStr string) (*Instance, error) {
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

	var config *rest.Config
	var err error

	// If kubeconfig string is provided, use it
	if kubeconfigStr != "" {
		config, err = kc.getConfigWithContext([]byte(kubeconfigStr), contextName)
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
		config, err = kc.buildConfigFromFileWithContext(kubeconfigPath, contextName)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// Create the client with our custom scheme
	c, err := client.New(config, client.Options{Scheme: customScheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Instance{
		client: c,
		config: instance,
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

// GetExporter retrieves a specific exporter by name
func (i *Instance) GetExporter(ctx context.Context, name string) (*v1alpha1.Exporter, error) {
	exporter := &v1alpha1.Exporter{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required to get exporter %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, exporter)
	return exporter, err
}

// UpdateExporter updates an exporter
func (i *Instance) UpdateExporter(ctx context.Context, exporter *v1alpha1.Exporter) error {
	return i.client.Update(ctx, exporter)
}

// CreateExporter creates a new exporter
func (i *Instance) CreateExporter(ctx context.Context, exporter *v1alpha1.Exporter) error {
	return i.client.Create(ctx, exporter)
}

// DeleteExporter deletes an exporter by name
func (i *Instance) DeleteExporter(ctx context.Context, name string) error {
	exporter := &v1alpha1.Exporter{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("namespace is required to delete exporter %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, exporter)
	if err != nil {
		return fmt.Errorf("failed to get exporter %s: %w", name, err)
	}

	return i.client.Delete(ctx, exporter)
}

// ListClients lists all clients in the instance's namespace
func (i *Instance) ListClients(ctx context.Context) (*v1alpha1.ClientList, error) {
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

// GetClientByName retrieves a specific client by name
func (i *Instance) GetClientByName(ctx context.Context, name string) (*v1alpha1.Client, error) {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required to get client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	return clientObj, err
}

// UpdateClient updates a client
func (i *Instance) UpdateClient(ctx context.Context, clientObj *v1alpha1.Client) error {
	return i.client.Update(ctx, clientObj)
}

// CreateClient creates a new client
func (i *Instance) CreateClient(ctx context.Context, clientObj *v1alpha1.Client) error {
	return i.client.Create(ctx, clientObj)
}

// DeleteClient deletes a client by name
func (i *Instance) DeleteClient(ctx context.Context, name string) error {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("namespace is required to delete client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	if err != nil {
		return fmt.Errorf("failed to get client %s: %w", name, err)
	}

	return i.client.Delete(ctx, clientObj)
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
