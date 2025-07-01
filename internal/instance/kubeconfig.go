package instance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeClient provides methods to create Kubernetes clients for accessing CRs
type KubeClient struct{}

// NewKubeClient creates a new KubeClient instance
func NewKubeClient() *KubeClient {
	return &KubeClient{}
}

// NewClientFromKubeconfigString creates a client from a kubeconfig string
func (kc *KubeClient) NewClientFromKubeconfigString(kubeconfigStr string) (client.Client, error) {
	return kc.NewClientFromKubeconfigStringWithContext(kubeconfigStr, "")
}

// NewClientFromKubeconfigStringWithContext creates a client from a kubeconfig string with a specific context
func (kc *KubeClient) NewClientFromKubeconfigStringWithContext(kubeconfigStr, contextName string) (client.Client, error) {
	// Parse the kubeconfig string
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig string: %w", err)
	}

	// If context name is provided, override the config
	if contextName != "" {
		config, err = kc.getConfigWithContext([]byte(kubeconfigStr), contextName)
		if err != nil {
			return nil, fmt.Errorf("failed to get config with context %s: %w", contextName, err)
		}
	}

	return kc.createClient(config)
}

// NewClientFromFile creates a client from a kubeconfig file path
func (kc *KubeClient) NewClientFromFile(kubeconfigPath string) (client.Client, error) {
	return kc.NewClientFromFileWithContext(kubeconfigPath, "")
}

// NewClientFromFileWithContext creates a client from a kubeconfig file path with a specific context
func (kc *KubeClient) NewClientFromFileWithContext(kubeconfigPath, contextName string) (client.Client, error) {
	// Expand home directory if needed
	if kubeconfigPath == "~/.kube/config" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Check if file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file does not exist: %s", kubeconfigPath)
	}

	// Load the kubeconfig with context
	config, err := kc.buildConfigFromFileWithContext(kubeconfigPath, contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	return kc.createClient(config)
}

// NewClientFromEnv creates a client using the KUBECONFIG environment variable or default locations
func (kc *KubeClient) NewClientFromEnv() (client.Client, error) {
	return kc.NewClientFromEnvWithContext("")
}

// NewClientFromEnvWithContext creates a client using the KUBECONFIG environment variable with a specific context
func (kc *KubeClient) NewClientFromEnvWithContext(contextName string) (client.Client, error) {
	// Get kubeconfig path from environment or use default
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	return kc.NewClientFromFileWithContext(kubeconfigPath, contextName)
}

// NewClientFromInCluster creates a client for in-cluster usage
func (kc *KubeClient) NewClientFromInCluster() (client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	return kc.createClient(config)
}

// NewClientFromConfig creates a client from an existing rest.Config
func (kc *KubeClient) NewClientFromConfig(config *rest.Config) (client.Client, error) {
	return kc.createClient(config)
}

// getConfigWithContext creates a rest.Config from kubeconfig bytes with a specific context
func (kc *KubeClient) getConfigWithContext(kubeconfigBytes []byte, contextName string) (*rest.Config, error) {
	// Load the kubeconfig
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create client config from bytes: %w", err)
	}

	// If context name is provided, we need to parse the kubeconfig and override
	if contextName != "" {
		// Parse the kubeconfig to get the raw config
		rawConfig, err := clientcmd.Load(kubeconfigBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}

		// Check if the context exists
		if _, exists := rawConfig.Contexts[contextName]; !exists {
			return nil, fmt.Errorf("context %s does not exist in kubeconfig", contextName)
		}

		// Create a new client config with the specified context
		clientConfig := clientcmd.NewNonInteractiveClientConfig(
			*rawConfig,
			contextName,
			&clientcmd.ConfigOverrides{},
			nil,
		)

		// Get the rest config
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get client config: %w", err)
		}
	}

	return config, nil
}

// buildConfigFromFileWithContext creates a rest.Config from a file with a specific context
func (kc *KubeClient) buildConfigFromFileWithContext(kubeconfigPath, contextName string) (*rest.Config, error) {
	// Load the kubeconfig file
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	)

	// Get the rest config
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return config, nil
}

// createClient is a helper method that creates a client with the proper scheme
func (kc *KubeClient) createClient(config *rest.Config) (client.Client, error) {
	// Add our API types to the scheme
	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	// Create the client
	c, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return c, nil
}

// Convenience functions for common use cases

// NewClient creates a client using the most appropriate method:
// 1. In-cluster config if running inside a pod
// 2. KUBECONFIG environment variable
// 3. Default ~/.kube/config
func NewClient() (client.Client, error) {
	return NewClientWithContext("")
}

// NewClientWithContext creates a client with a specific context
func NewClientWithContext(contextName string) (client.Client, error) {
	kc := NewKubeClient()

	// Try in-cluster first (context doesn't apply to in-cluster)
	if c, err := kc.NewClientFromInCluster(); err == nil {
		return c, nil
	}

	// Fall back to environment/file-based config with context
	return kc.NewClientFromEnvWithContext(contextName)
}

// NewClientWithGoContext creates a client with a Go context (for future use)
func NewClientWithGoContext(ctx context.Context) (client.Client, error) {
	return NewClient()
}
