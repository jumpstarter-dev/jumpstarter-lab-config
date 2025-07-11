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

// GetExporterByName retrieves a specific exporter by name
func (i *Instance) GetExporterByName(ctx context.Context, name string) (*v1alpha1.Exporter, error) {
	return i.getExporterByName(ctx, name)
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

// checkAndPrintDiff prints a diff between two objects, ignoring Kubernetes metadata fields
func (i *Instance) checkAndPrintDiff(oldObj, newObj interface{}, objType, objName string) bool {
	// Options to ignore Kubernetes metadata fields that change frequently
	ignoreOpts := []cmp.Option{
		cmpopts.IgnoreFields(metav1.ObjectMeta{}, "Generation", "CreationTimestamp", "ResourceVersion", "UID", "ManagedFields"),
		cmpopts.IgnoreFields(v1alpha1.Exporter{}, "Status"),
		cmpopts.IgnoreFields(v1alpha1.Client{}, "Status"),
	}

	diff := cmp.Diff(oldObj, newObj, ignoreOpts...)
	if diff != "" {
		fmt.Printf("📝 [%s] dry run: Would update %s %s, diff: %s\n", i.config.Name, objType, objName, diff)
		return true
	}

	fmt.Printf("✅ [%s] dry run: No changes needed for %s %s\n", i.config.Name, objType, objName)
	return false
}
