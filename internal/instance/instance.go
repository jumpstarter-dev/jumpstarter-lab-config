package instance

import (
	"fmt"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewKubeClientFromInstance creates a Kubernetes client from a JumpstarterInstance and optional kubeconfig string
// If kubeconfigStr is empty, it will try to load from environment/standard kubeconfig file
func NewKubeClientFromInstance(instance *v1alpha1.JumpstarterInstance, kubeconfigStr string) (client.Client, error) {
	// Validate the instance
	if err := validateInstance(instance); err != nil {
		return nil, fmt.Errorf("invalid instance: %w", err)
	}

	// Get the context from the instance
	contextName := instance.Spec.KubeContext

	// Create a kube client
	kc := NewKubeClient()

	// If kubeconfig string is provided, use it
	if kubeconfigStr != "" {
		return kc.NewClientFromKubeconfigStringWithContext(kubeconfigStr, contextName)
	}

	// Otherwise, try to load from environment/standard kubeconfig file
	return kc.NewClientFromEnvWithContext(contextName)
}

// validateInstance performs basic validation on a JumpstarterInstance
func validateInstance(instance *v1alpha1.JumpstarterInstance) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	// Additional validation can be added here as needed
	// For example, checking if required fields are present

	return nil
}
