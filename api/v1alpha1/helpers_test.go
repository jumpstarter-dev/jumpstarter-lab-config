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

func TestExporterInstance_IsUnmanaged(t *testing.T) {
	tests := []struct {
		name          string
		instance      *ExporterInstance
		expectedState bool
		expectedValue string
	}{
		{
			name:          "nil instance",
			instance:      nil,
			expectedState: false,
			expectedValue: "",
		},
		{
			name: "no annotations",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
				},
			},
			expectedState: false,
			expectedValue: "",
		},
		{
			name: "managed exporter with other annotation",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
					Annotations: map[string]string{
						"example": "value",
					},
				},
			},
			expectedState: false,
			expectedValue: "",
		},
		{
			name: "unmanaged exporter with empty discovery date",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
					Annotations: map[string]string{
						UnmanagedAnnotation: "",
					},
				},
			},
			expectedState: true,
			expectedValue: "",
		},
		{
			name: "unmanaged exporter with discovery date",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-exporter",
					Annotations: map[string]string{
						UnmanagedAnnotation: "2026-02-01",
					},
				},
			},
			expectedState: true,
			expectedValue: "2026-02-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUnmanaged, value := tt.instance.IsUnmanaged()
			assert.Equal(t, tt.expectedState, isUnmanaged)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestExporterInstance_IsDead(t *testing.T) {
	tests := []struct {
		name          string
		instance      *ExporterInstance
		expectedState bool
		expectedValue string
	}{
		{
			name:          "nil instance",
			instance:      nil,
			expectedState: false,
			expectedValue: "",
		},
		{
			name: "no annotations",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "exporter"},
			},
			expectedState: false,
			expectedValue: "",
		},
		{
			name: "dead via current annotation key",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exporter",
					Annotations: map[string]string{
						DeadAnnotation: "maintenance",
					},
				},
			},
			expectedState: true,
			expectedValue: "maintenance",
		},
		{
			name: "dead via legacy annotation key",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exporter",
					Annotations: map[string]string{
						LegacyDeadAnnotation: "legacy-maintenance",
					},
				},
			},
			expectedState: true,
			expectedValue: "legacy-maintenance",
		},
		{
			name: "dead annotation takes precedence over legacy annotation",
			instance: &ExporterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exporter",
					Annotations: map[string]string{
						DeadAnnotation:       "current-maintenance",
						LegacyDeadAnnotation: "legacy-maintenance",
					},
				},
			},
			expectedState: true,
			expectedValue: "current-maintenance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDead, value := tt.instance.IsDead()
			assert.Equal(t, tt.expectedState, isDead)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}
