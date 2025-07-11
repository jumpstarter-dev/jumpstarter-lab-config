package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExporterInstance_HasConfigTemplate(t *testing.T) {
	tests := []struct {
		name     string
		instance *ExporterInstance
		expected bool
	}{
		{
			name: "has config template",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter-with-template",
				},
				Spec: ExporterInstanceSpec{
					ConfigTemplateRef: ConfigTemplateRef{
						Name: "test-template",
					},
				},
			},
			expected: true,
		},
		{
			name: "no config template - empty name",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter-no-template",
				},
				Spec: ExporterInstanceSpec{
					ConfigTemplateRef: ConfigTemplateRef{
						Name: "",
					},
				},
			},
			expected: false,
		},
		{
			name: "no config template - uninitialized",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter-uninitialized",
				},
				Spec: ExporterInstanceSpec{
					// ConfigTemplateRef is not initialized, so Name should be empty
				},
			},
			expected: false,
		},
		{
			name: "config template with whitespace only",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter-whitespace",
				},
				Spec: ExporterInstanceSpec{
					ConfigTemplateRef: ConfigTemplateRef{
						Name: "   ",
					},
				},
			},
			expected: true, // whitespace is considered a valid name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.instance.HasConfigTemplate()
			assert.Equal(t, tt.expected, result, "HasConfigTemplate() should return %v for case %s", tt.expected, tt.name)
		})
	}
}

func TestExporterInstance_HasConfigTemplate_EdgeCases(t *testing.T) {
	t.Run("nil instance", func(t *testing.T) {
		var instance *ExporterInstance
		// This should panic, but let's test it doesn't unexpectedly crash
		assert.Panics(t, func() {
			instance.HasConfigTemplate()
		}, "HasConfigTemplate() should panic when called on nil instance")
	})

	t.Run("valid template name", func(t *testing.T) {
		instance := &ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-exporter",
			},
			Spec: ExporterInstanceSpec{
				ConfigTemplateRef: ConfigTemplateRef{
					Name: "my-template-123",
				},
			},
		}
		assert.True(t, instance.HasConfigTemplate(), "HasConfigTemplate() should return true for valid template name")
	})

	t.Run("template with special characters", func(t *testing.T) {
		instance := &ExporterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-exporter",
			},
			Spec: ExporterInstanceSpec{
				ConfigTemplateRef: ConfigTemplateRef{
					Name: "template-with-dashes_and_underscores.123",
				},
			},
		}
		assert.True(t, instance.HasConfigTemplate(), "HasConfigTemplate() should return true for template name with special characters")
	})
}
