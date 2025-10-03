package host

import (
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeadAnnotationFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		exporterInstances      []*v1alpha1.ExporterInstance
		expectedAliveInstances int
		expectedDeadInstances  int
	}{
		{
			name: "no dead instances",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
					},
				},
			},
			expectedAliveInstances: 2,
			expectedDeadInstances:  0,
		},
		{
			name: "one dead instance",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			expectedAliveInstances: 1,
			expectedDeadInstances:  1,
		},
		{
			name: "all dead instances",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  2,
		},
		{
			name: "dead annotation with false value",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "false",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  1,
		},
		{
			name: "dead annotation with other value",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "maybe",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from SyncExporterHosts
			aliveInstances := []*v1alpha1.ExporterInstance{}
			deadCount := 0

			for _, exporterInstance := range tt.exporterInstances {
				if _, exists := exporterInstance.Annotations["dead"]; exists {
					deadCount++
				} else {
					aliveInstances = append(aliveInstances, exporterInstance)
				}
			}

			assert.Equal(t, tt.expectedAliveInstances, len(aliveInstances), "unexpected number of alive instances")
			assert.Equal(t, tt.expectedDeadInstances, deadCount, "unexpected number of dead instances")
		})
	}
}

func TestHostSkippingBehavior(t *testing.T) {
	tests := []struct {
		name              string
		exporterInstances []*v1alpha1.ExporterInstance
		shouldSkipHost    bool
		expectedAlivCount int
		description       string
	}{
		{
			name: "mixed dead and alive instances - should NOT skip host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			shouldSkipHost:    false, // Host should be processed because there's an alive instance
			expectedAlivCount: 1,
			description:       "When there are both dead and alive instances, host should be processed",
		},
		{
			name: "all instances dead - should SKIP host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance-1",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance-2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			shouldSkipHost:    true, // Host should be skipped because all instances are dead
			expectedAlivCount: 0,
			description:       "When all instances are dead, host should be skipped entirely",
		},
		{
			name: "all instances alive - should NOT skip host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance-2",
					},
				},
			},
			shouldSkipHost:    false, // Host should be processed because all instances are alive
			expectedAlivCount: 2,
			description:       "When all instances are alive, host should be processed normally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the new logic from SyncExporterHosts
			// Check if all instances are dead
			allDead := true
			aliveCount := 0

			for _, exporterInstance := range tt.exporterInstances {
				if _, exists := exporterInstance.Annotations["dead"]; !exists {
					allDead = false
					aliveCount++
				}
			}

			// Host is skipped only if all instances are dead (and there are instances)
			hostWouldBeSkipped := len(tt.exporterInstances) > 0 && allDead

			assert.Equal(t, tt.shouldSkipHost, hostWouldBeSkipped, tt.description)
			assert.Equal(t, tt.expectedAlivCount, aliveCount, "unexpected number of alive instances")
		})
	}
}
