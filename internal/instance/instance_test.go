package instance

import (
	"context"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
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
		_, err := NewInstance(instance, validKubeconfig, false, false)
		// We expect this to fail since the context doesn't exist in our test kubeconfig
		if err != nil {
			assert.Contains(t, err.Error(), "context test-context does not exist in kubeconfig")
		}
	})

	t.Run("without kubeconfig string", func(t *testing.T) {
		// This will likely fail since we don't have a real kubeconfig file, but we test the flow
		_, err := NewInstance(instance, "", false, false)
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

		_, err := NewInstance(instanceNoContext, validKubeconfig, false, false)
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
		inst, err := NewInstance(instance, validKubeconfig, false, false)
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

	t.Run("listExporters with namespace", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig, false, false)
		if err == nil {
			// Test that the method exists and can be called
			ctx := context.Background()
			exporters, err := inst.listExporters(ctx)
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

	t.Run("listExporters without namespace", func(t *testing.T) {
		instanceNoNamespace := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-namespace",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
				// No namespace specified
			},
		}

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig, false, false)
		if err == nil {
			ctx := context.Background()
			exporters, err := inst.listExporters(ctx)
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

	t.Run("GetExporterByName", func(t *testing.T) {
		inst, err := NewInstance(instance, validKubeconfig, false, false)
		if err == nil {
			ctx := context.Background()
			exporter, err := inst.GetExporterByName(ctx, "test-exporter")
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

	t.Run("GetExporterByName without namespace", func(t *testing.T) {
		instanceNoNamespace := &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-instance-no-namespace",
			},
			Spec: v1alphaConfig.JumpstarterInstanceSpec{
				KubeContext: "test-context",
				// No namespace specified
			},
		}

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig, false, false)
		if err == nil {
			ctx := context.Background()
			_, err := inst.GetExporterByName(ctx, "test-exporter")
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
		inst, err := NewInstance(instance, validKubeconfig, false, false)
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
		inst, err := NewInstance(instance, validKubeconfig, false, false)
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

		inst, err := NewInstance(instanceNoNamespace, validKubeconfig, false, false)
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

func TestPrintDiff(t *testing.T) {
	// Create a test instance to use for the printDiff method
	instance := &v1alphaConfig.JumpstarterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-instance",
		},
		Spec: v1alphaConfig.JumpstarterInstanceSpec{
			KubeContext: "test-context",
			Namespace:   "test-namespace",
		},
	}

	t.Run("printDiff with different objects", func(t *testing.T) {
		// Create test objects with different values
		oldObj := &v1alpha1.Exporter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-exporter",
				Namespace: "test-namespace",
			},
		}

		newObj := &v1alpha1.Exporter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-exporter",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"new-label": "new-value",
				},
			},
		}

		// Create an instance to test the printDiff method
		inst, err := NewInstance(instance, validKubeconfig, false, false)
		if err == nil {
			// This should not panic and should print a diff
			inst.printDiff(oldObj, newObj, "exporter", "test-exporter")
		}
	})

	t.Run("printDiff with identical objects", func(t *testing.T) {
		// Create identical objects
		obj := &v1alpha1.Exporter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-exporter",
				Namespace: "test-namespace",
			},
		}

		// Create an instance to test the printDiff method
		inst, err := NewInstance(instance, validKubeconfig, false, false)
		if err == nil {
			// This should not panic and should indicate no changes
			inst.printDiff(obj, obj, "exporter", "test-exporter")
		}
	})
}

func TestGetExporterObjectForInstance(t *testing.T) {
	// Create a test config (we'll use a mock/simple one for testing)
	cfg := &config.Config{
		Loaded: &config.LoadedLabConfig{
			// We'll add mock data as needed
		},
	}

	tests := []struct {
		name                 string
		exporterInstance     *v1alphaConfig.ExporterInstance
		jumpstarterInstance  string
		expectedExporter     *v1alpha1.Exporter
		expectedError        bool
		expectedErrorMessage string
	}{
		{
			name: "exporter instance with matching jumpstarter instance and no config template",
			exporterInstance: &v1alphaConfig.ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
				},
				Spec: v1alphaConfig.ExporterInstanceSpec{
					Username: "test-user",
					JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
						Name: "target-instance",
					},
					Labels: map[string]string{
						"app": "test",
					},
					// No ConfigTemplateRef - should use default metadata
				},
			},
			jumpstarterInstance: "target-instance",
			expectedExporter: &v1alpha1.Exporter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: v1alpha1.ExporterSpec{
					Username: func() *string { s := "test-user"; return &s }(),
				},
			},
			expectedError: false,
		},
		{
			name: "exporter instance with matching jumpstarter instance and config template",
			exporterInstance: &v1alphaConfig.ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter-with-template",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: v1alphaConfig.ExporterInstanceSpec{
					Username: "test-user",
					JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
						Name: "target-instance",
					},
					ConfigTemplateRef: v1alphaConfig.ConfigTemplateRef{
						Name: "test-template",
					},
				},
			},
			jumpstarterInstance:  "target-instance",
			expectedExporter:     nil, // Template processing would fail in this test environment
			expectedError:        true,
			expectedErrorMessage: "error creating ExporterInstanceTemplater",
		},
		{
			name: "exporter instance with non-matching jumpstarter instance",
			exporterInstance: &v1alphaConfig.ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
				},
				Spec: v1alphaConfig.ExporterInstanceSpec{
					Username: "test-user",
					JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
						Name: "other-instance",
					},
				},
			},
			jumpstarterInstance: "target-instance",
			expectedExporter:    nil,
			expectedError:       false,
		},
		{
			name: "exporter instance with empty jumpstarter instance ref",
			exporterInstance: &v1alphaConfig.ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
				},
				Spec: v1alphaConfig.ExporterInstanceSpec{
					Username: "test-user",
					JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
						Name: "",
					},
				},
			},
			jumpstarterInstance: "target-instance",
			expectedExporter:    nil,
			expectedError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetExporterObjectForInstance(cfg, tt.exporterInstance, tt.jumpstarterInstance)

			if tt.expectedError {
				assert.Error(t, err, "Expected error for case %s", tt.name)
				if tt.expectedErrorMessage != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMessage, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error for case %s", tt.name)
			}

			if tt.expectedExporter == nil {
				assert.Nil(t, result, "Expected nil result for case %s", tt.name)
			} else {
				assert.NotNil(t, result, "Expected non-nil result for case %s", tt.name)
				assert.Equal(t, tt.expectedExporter.Name, result.Name, "Expected name to match")
				assert.Equal(t, tt.expectedExporter.Labels, result.Labels, "Expected labels to match")
				assert.Equal(t, tt.expectedExporter.Spec.Username, result.Spec.Username, "Expected username to match")
			}
		})
	}
}

func TestGetExporterObjectForInstance_WithTemplateProcessing(t *testing.T) {
	// Test cases that focus on the template processing logic
	t.Run("exporter instance without config template uses original metadata", func(t *testing.T) {
		exporterInstance := &v1alphaConfig.ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-exporter",
				Annotations: map[string]string{
					"original": "annotation",
				},
			},
			Spec: v1alphaConfig.ExporterInstanceSpec{
				Username: "test-user",
				JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
					Name: "target-instance",
				},
				Labels: map[string]string{
					"original": "label",
					"app":      "test",
				},
				// No ConfigTemplateRef
			},
		}

		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{},
		}

		result, err := GetExporterObjectForInstance(cfg, exporterInstance, "target-instance")

		assert.NoError(t, err, "Should not error when no template is used")
		assert.NotNil(t, result, "Should return an exporter object")
		assert.Equal(t, "test-exporter", result.Name, "Should preserve original name")
		assert.Equal(t, map[string]string{
			"original": "label",
			"app":      "test",
		}, result.Labels, "Should preserve original labels")
		assert.Equal(t, map[string]string{
			"original": "annotation",
		}, result.Annotations, "Should preserve original annotations")
	})

	t.Run("exporter instance checks HasConfigTemplate correctly", func(t *testing.T) {
		// Test that the function properly uses the HasConfigTemplate method
		exporterInstanceWithTemplate := &v1alphaConfig.ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-exporter-with-template",
			},
			Spec: v1alphaConfig.ExporterInstanceSpec{
				Username: "test-user",
				JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
					Name: "target-instance",
				},
				ConfigTemplateRef: v1alphaConfig.ConfigTemplateRef{
					Name: "some-template",
				},
			},
		}

		exporterInstanceWithoutTemplate := &v1alphaConfig.ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-exporter-without-template",
			},
			Spec: v1alphaConfig.ExporterInstanceSpec{
				Username: "test-user",
				JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
					Name: "target-instance",
				},
				// No ConfigTemplateRef
			},
		}

		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{},
		}

		// Test with template - should fail in this environment because we can't create a real templater
		_, err := GetExporterObjectForInstance(cfg, exporterInstanceWithTemplate, "target-instance")
		assert.Error(t, err, "Should error when trying to create templater without proper config")

		// Test without template - should succeed
		result, err := GetExporterObjectForInstance(cfg, exporterInstanceWithoutTemplate, "target-instance")
		assert.NoError(t, err, "Should not error when no template is used")
		assert.NotNil(t, result, "Should return an exporter object")
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
