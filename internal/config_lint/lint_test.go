package config_lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
)

func TestValidateTemplates(t *testing.T) {
	tests := []struct {
		name                    string
		exporterInstances       map[string]*v1alphaConfig.ExporterInstance
		exporterConfigTemplates map[string]*v1alphaConfig.ExporterConfigTemplate
		expectedErrors          map[string]int // map of error keys to expected count
	}{
		{
			name: "exporter instance without config template - should pass",
			exporterInstances: map[string]*v1alphaConfig.ExporterInstance{
				"test-exporter-no-template": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-exporter-no-template",
					},
					Spec: v1alphaConfig.ExporterInstanceSpec{
						Username: "test-user",
						JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
							Name: "test-instance",
						},
						// No ConfigTemplateRef
					},
				},
			},
			exporterConfigTemplates: map[string]*v1alphaConfig.ExporterConfigTemplate{},
			expectedErrors:          map[string]int{},
		},
		{
			name: "exporter instance with config template but no template defined - should fail",
			exporterInstances: map[string]*v1alphaConfig.ExporterInstance{
				"test-exporter-with-template": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-exporter-with-template",
					},
					Spec: v1alphaConfig.ExporterInstanceSpec{
						Username: "test-user",
						JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
							Name: "test-instance",
						},
						ConfigTemplateRef: v1alphaConfig.ConfigTemplateRef{
							Name: "missing-template",
						},
					},
				},
			},
			exporterConfigTemplates: map[string]*v1alphaConfig.ExporterConfigTemplate{},
			expectedErrors:          map[string]int{"test-exporter-with-template": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test config
			cfg := &config.Config{
				Loaded: &config.LoadedLabConfig{
					ExporterInstances:       tt.exporterInstances,
					ExporterConfigTemplates: tt.exporterConfigTemplates,
				},
			}

			// Call the function under test
			errorsByItem := validateTemplates(cfg)

			// Verify the expected errors
			for expectedKey, expectedCount := range tt.expectedErrors {
				errors, exists := errorsByItem[expectedKey]
				assert.True(t, exists, "Expected error key %s to exist", expectedKey)
				assert.Len(t, errors, expectedCount, "Expected %d errors for key %s, got %d", expectedCount, expectedKey, len(errors))
			}

			// Verify no unexpected errors
			for actualKey := range errorsByItem {
				_, expected := tt.expectedErrors[actualKey]
				assert.True(t, expected, "Unexpected error key %s found", actualKey)
			}
		})
	}
}

func TestValidateTemplates_HasConfigTemplateUsage(t *testing.T) {
	t.Run("validates that HasConfigTemplate method is used correctly", func(t *testing.T) {
		// Create exporter instances with different template configurations
		exporterWithTemplate := &v1alphaConfig.ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "exporter-with-template",
			},
			Spec: v1alphaConfig.ExporterInstanceSpec{
				Username: "test-user",
				JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
					Name: "test-instance",
				},
				ConfigTemplateRef: v1alphaConfig.ConfigTemplateRef{
					Name: "some-template",
				},
			},
		}

		exporterWithoutTemplate := &v1alphaConfig.ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "exporter-without-template",
			},
			Spec: v1alphaConfig.ExporterInstanceSpec{
				Username: "test-user",
				JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
					Name: "test-instance",
				},
				// No ConfigTemplateRef
			},
		}

		// Test that the lint function processes only the exporter with template
		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{
				ExporterInstances: map[string]*v1alphaConfig.ExporterInstance{
					"exporter-with-template":    exporterWithTemplate,
					"exporter-without-template": exporterWithoutTemplate,
				},
				ExporterConfigTemplates: map[string]*v1alphaConfig.ExporterConfigTemplate{},
			},
		}

		errorsByItem := validateTemplates(cfg)

		// Only the exporter with template should have errors (since template is missing)
		assert.Contains(t, errorsByItem, "exporter-with-template", "Expected error for exporter with template")
		assert.NotContains(t, errorsByItem, "exporter-without-template", "Should not have error for exporter without template")
	})
}

func TestValidateTemplates_EdgeCases(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{
				ExporterInstances:       map[string]*v1alphaConfig.ExporterInstance{},
				ExporterConfigTemplates: map[string]*v1alphaConfig.ExporterConfigTemplate{},
			},
		}

		errorsByItem := validateTemplates(cfg)
		assert.Empty(t, errorsByItem, "Empty config should not produce errors")
	})

	t.Run("config with nil fields", func(t *testing.T) {
		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{
				ExporterInstances:       nil,
				ExporterConfigTemplates: nil,
			},
		}

		// This should handle nil maps gracefully
		errorsByItem := validateTemplates(cfg)
		assert.NotNil(t, errorsByItem, "Should return non-nil error map")
		assert.Empty(t, errorsByItem, "Should not produce errors for nil maps")
	})
}

func TestLint(t *testing.T) {
	t.Run("integration test with lint function", func(t *testing.T) {
		// Create a test config with both template and non-template exporters
		cfg := &config.Config{
			Loaded: &config.LoadedLabConfig{
				ExporterInstances: map[string]*v1alphaConfig.ExporterInstance{
					"valid-exporter-no-template": {
						ObjectMeta: metav1.ObjectMeta{
							Name: "valid-exporter-no-template",
						},
						Spec: v1alphaConfig.ExporterInstanceSpec{
							Username: "test-user",
							JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
								Name: "test-instance",
							},
							// No ConfigTemplateRef
						},
					},
					"invalid-exporter-with-template": {
						ObjectMeta: metav1.ObjectMeta{
							Name: "invalid-exporter-with-template",
						},
						Spec: v1alphaConfig.ExporterInstanceSpec{
							Username: "test-user",
							JumpstarterInstanceRef: v1alphaConfig.JumsptarterInstanceRef{
								Name: "test-instance",
							},
							ConfigTemplateRef: v1alphaConfig.ConfigTemplateRef{
								Name: "missing-template",
							},
						},
					},
				},
				ExporterConfigTemplates: map[string]*v1alphaConfig.ExporterConfigTemplate{},
			},
		}

		// Run the full lint process
		errorsByItem := Lint(cfg)

		// Check that we get errors for the invalid exporter but not the valid one
		validExporterErrors, hasValidExporterError := errorsByItem["valid-exporter-no-template"]
		invalidExporterErrors, hasInvalidExporterError := errorsByItem["invalid-exporter-with-template"]

		assert.False(t, hasValidExporterError || len(validExporterErrors) > 0, "Should not have errors for valid exporter without template")
		assert.True(t, hasInvalidExporterError && len(invalidExporterErrors) > 0, "Should have errors for invalid exporter with missing template")
	})
}
