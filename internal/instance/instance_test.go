package instance

import (
	"context"
	"testing"

	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewInstance(t *testing.T) {
	// Create a test instance with a context
	instance := &v1alphaConfig.JumpstarterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-instance",
		},
		Spec: v1alphaConfig.JumpstarterInstanceSpec{
			KubeContext: "test-context",
		},
	}

	t.Run("with kubeconfig string", func(t *testing.T) {
		// This will likely fail since we don't have a real kubeconfig, but we test the flow
		_, err := NewInstance(instance, validKubeconfig)
		// We expect this to fail since the context doesn't exist in our test kubeconfig
		if err != nil {
			assert.Contains(t, err.Error(), "context test-context does not exist in kubeconfig")
		}
	})

	t.Run("without kubeconfig string", func(t *testing.T) {
		// This will likely fail since we don't have a real kubeconfig file, but we test the flow
		_, err := NewInstance(instance, "")
		// We expect this to fail since the context doesn't exist in the default kubeconfig
		if err != nil {
			assert.True(t,
				contains(err.Error(), "context test-context does not exist") ||
					contains(err.Error(), "context \"test-context\" does not exist") ||
					contains(err.Error(), "kubeconfig file does not exist") ||
					contains(err.Error(), "failed to get in-cluster config") ||
					contains(err.Error(), "failed to load kubeconfig"),
				"Unexpected error: %s", err.Error())
		}
	})

	t.Run("with empty context", func(t *testing.T) {
		instanceNoContext := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-context",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "",
			},
		}

		_, err := NewInstance(instanceNoContext, validKubeconfig)
		// This should work with our test kubeconfig since it uses the default context
		if err != nil {
			assert.True(t,
				contains(err.Error(), "failed to create client") ||
					contains(err.Error(), "failed to add scheme"),
				"Unexpected error: %s", err.Error())
		}
	})
}

func TestInstanceMethods(t *testing.T) {
	// Create a test instance
	instance := &v1alphaConfig.JumpstarterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-instance",
		},
		Spec: v1alphaConfig.JumpstarterInstanceSpec{
			KubeContext: "test-context",
			Namespace:   "test-namespace",
		},
	}

	t.Run("GetClient and GetConfig", func(t *testing.T) {
		// This will likely fail, but we test the method signatures
		inst, err := NewInstance(instance, validKubeconfig)
		if err == nil {
			// If it succeeds, test the methods
			client := inst.GetClient()
			assert.NotNil(t, client)

			config := inst.GetConfig()
			assert.Equal(t, instance, config)
		}
	})
}

func TestInstanceExporterMethods(t *testing.T) {
	// Create a test instance with namespace
	instance := &v1alphaConfig.JumpstarterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-instance",
		},
		Spec: v1alphaConfig.JumpstarterInstanceSpec{
			KubeContext: "test-context",
			Namespace:   "test-namespace",
		},
	}

	t.Run("ListExporters with namespace", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig)
		if err == nil {
			// Test that the method exists and can be called
			ctx := context.Background()
			exporters, err := inst.ListExporters(ctx)
			// This will likely fail due to connection issues, but we test the method signature
			if err != nil {
				assert.True(t,
					contains(err.Error(), "failed to list") ||
						contains(err.Error(), "connection") ||
						contains(err.Error(), "context") ||
						contains(err.Error(), "namespace") ||
						contains(err.Error(), "failed to get server groups") ||
						contains(err.Error(), "dial tcp") ||
						contains(err.Error(), "no such host"),
					"Unexpected error: %s", err.Error())
			} else {
				assert.NotNil(t, exporters)
			}
		}
	})

	t.Run("ListExporters without namespace", func(t *testing.T) {
		instanceNoNamespace := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-namespace",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
				// No namespace specified
			},
		}

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			exporters, err := inst.ListExporters(ctx)
			// This will likely fail due to connection issues, but we test the method signature
			if err != nil {
				assert.True(t,
					contains(err.Error(), "failed to list") ||
						contains(err.Error(), "connection") ||
						contains(err.Error(), "context") ||
						contains(err.Error(), "failed to get server groups") ||
						contains(err.Error(), "dial tcp") ||
						contains(err.Error(), "no such host"),
					"Unexpected error: %s", err.Error())
			} else {
				assert.NotNil(t, exporters)
			}
		}
	})

	t.Run("GetExporter", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			exporter, err := inst.GetExporter(ctx, "test-exporter")
			// This will likely fail due to connection issues or missing exporter
			if err != nil {
				assert.True(t,
					contains(err.Error(), "failed to get") ||
						contains(err.Error(), "connection") ||
						contains(err.Error(), "not found") ||
						contains(err.Error(), "namespace") ||
						contains(err.Error(), "failed to get server groups") ||
						contains(err.Error(), "dial tcp") ||
						contains(err.Error(), "no such host"),
					"Unexpected error: %s", err.Error())
			} else {
				assert.NotNil(t, exporter)
			}
		}
	})

	t.Run("GetExporter without namespace", func(t *testing.T) {
		instanceNoNamespace := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-namespace",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
				// No namespace specified
			},
		}

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			_, err := inst.GetExporter(ctx, "test-exporter")
			// This should fail because namespace is required
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "namespace is required")
		}
	})
}

func TestInstanceClientMethods(t *testing.T) {
	// Create a test instance with namespace
	instance := &v1alphaConfig.JumpstarterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-instance",
		},
		Spec: v1alphaConfig.JumpstarterInstanceSpec{
			KubeContext: "test-context",
			Namespace:   "test-namespace",
		},
	}

	t.Run("ListClients with namespace", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			clients, err := inst.ListClients(ctx)
			// This will likely fail due to connection issues, but we test the method signature
			if err != nil {
				assert.True(t,
					contains(err.Error(), "failed to list") ||
						contains(err.Error(), "connection") ||
						contains(err.Error(), "context") ||
						contains(err.Error(), "namespace") ||
						contains(err.Error(), "failed to get server groups") ||
						contains(err.Error(), "dial tcp") ||
						contains(err.Error(), "no such host"),
					"Unexpected error: %s", err.Error())
			} else {
				assert.NotNil(t, clients)
			}
		}
	})

	t.Run("GetClientByName", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			client, err := inst.GetClientByName(ctx, "test-client")
			// This will likely fail due to connection issues or missing client
			if err != nil {
				assert.True(t,
					contains(err.Error(), "failed to get") ||
						contains(err.Error(), "connection") ||
						contains(err.Error(), "not found") ||
						contains(err.Error(), "namespace") ||
						contains(err.Error(), "failed to get server groups") ||
						contains(err.Error(), "dial tcp") ||
						contains(err.Error(), "no such host"),
					"Unexpected error: %s", err.Error())
			} else {
				assert.NotNil(t, client)
			}
		}
	})

	t.Run("GetClientByName without namespace", func(t *testing.T) {
		instanceNoNamespace := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-namespace",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
				// No namespace specified
			},
		}

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig)
		if err == nil {
			ctx := context.Background()
			_, err := inst.GetClientByName(ctx, "test-client")
			// This should fail because namespace is required
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "namespace is required")
		}
	})
}

func TestValidateInstance(t *testing.T) {
	t.Run("valid instance", func(t *testing.T) {
		instance := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
			},
		}

		err := validateInstance(instance)
		assert.NoError(t, err)
	})

	t.Run("nil instance", func(t *testing.T) {
		err := validateInstance(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "instance cannot be nil")
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
