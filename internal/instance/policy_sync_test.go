package instance

import (
	"context"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSyncPolicies(t *testing.T) {
	for _, tt := range []struct {
		name   string
		dryRun bool
		prune  bool
	}{
		{name: "apply"},
		{name: "prune", prune: true},
		{name: "dry run", dryRun: true},
		{name: "dry run with prune", dryRun: true, prune: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			existing := &v1alpha1.ExporterAccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing", Namespace: pruneTestNamespace,
					Labels:      map[string]string{"old": "label"},
					Annotations: map[string]string{"preserved": "annotation", "updated": "old"},
				},
				Spec: v1alpha1.ExporterAccessPolicySpec{Policies: []v1alpha1.Policy{{Priority: 1}}},
			}
			stale := &v1alpha1.ExporterAccessPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "stale", Namespace: pruneTestNamespace},
			}
			outside := stale.DeepCopy()
			outside.Namespace = "other-lab"
			inst := newPruneTestInstance(t, tt.dryRun, tt.prune, existing, stale, outside)
			desired := &v1alpha1.ExporterAccessPolicy{
				TypeMeta: metav1.TypeMeta{APIVersion: "jumpstarter.dev/v1alpha1", Kind: "ExporterAccessPolicy"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing", Namespace: "manifest-namespace",
					Labels:      map[string]string{"new": "label"},
					Annotations: map[string]string{"updated": "new"},
				},
				Spec: v1alpha1.ExporterAccessPolicySpec{
					ExporterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"purpose": "dev"}},
					Policies: []v1alpha1.Policy{{Priority: 10, SpotAccess: true,
						From: []v1alpha1.From{{ClientSelector: metav1.LabelSelector{MatchLabels: map[string]string{"team": "dev"}}}}}},
				},
			}
			newPolicy := desired.DeepCopy()
			newPolicy.Name = "new"
			newPolicy.Namespace = ""
			cfg := &config.Config{Loaded: &config.LoadedLabConfig{
				Policies: map[string]*v1alpha1.ExporterAccessPolicy{"existing": desired, "new": newPolicy},
			}}
			original, originalNew := desired.DeepCopy(), newPolicy.DeepCopy()

			require.NoError(t, inst.SyncPolicies(ctx, cfg))
			got := &v1alpha1.ExporterAccessPolicy{}
			require.NoError(t, inst.client.Get(ctx, client.ObjectKeyFromObject(existing), got))
			if tt.dryRun {
				assert.Equal(t, existing.Spec, got.Spec)
				assert.Equal(t, existing.Labels, got.Labels)
				assert.Equal(t, existing.Annotations, got.Annotations)
			} else {
				assert.Equal(t, desired.Spec, got.Spec)
				assert.Equal(t, desired.Labels, got.Labels)
				assert.Equal(t, map[string]string{
					"preserved": "annotation", "updated": "new", "managed-by": managedByAnnotation,
				}, got.Annotations)
			}
			version := got.ResourceVersion
			err := inst.client.Get(ctx, client.ObjectKey{Namespace: pruneTestNamespace, Name: "new"}, got)
			if tt.dryRun {
				require.True(t, apierrors.IsNotFound(err), "unexpected Get result: %v", err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, newPolicy.Spec, got.Spec)
				assert.Equal(t, managedByAnnotation, got.Annotations["managed-by"])
			}
			err = inst.client.Get(ctx, client.ObjectKeyFromObject(stale), got)
			if tt.prune && !tt.dryRun {
				require.True(t, apierrors.IsNotFound(err), "unexpected Get result: %v", err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, inst.client.Get(ctx, client.ObjectKeyFromObject(outside), got))

			// Reapplying is a no-op, and the shared configuration remains reusable.
			require.NoError(t, inst.SyncPolicies(ctx, cfg))
			require.NoError(t, inst.client.Get(ctx, client.ObjectKeyFromObject(existing), got))
			assert.Equal(t, version, got.ResourceVersion)
			inst.config.Spec.Namespace = "second-lab"
			require.NoError(t, inst.SyncPolicies(ctx, cfg))
			if !tt.dryRun {
				require.NoError(t, inst.client.Get(ctx, client.ObjectKey{Namespace: "second-lab", Name: "existing"}, got))
				assert.Equal(t, desired.Spec, got.Spec)
			}
			assert.Equal(t, original, desired)
			assert.Equal(t, originalNew, newPolicy)
		})
	}
}

func TestSyncPoliciesEmptyConfig(t *testing.T) {
	policy := &v1alpha1.ExporterAccessPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "stale", Namespace: pruneTestNamespace},
	}
	inst := newPruneTestInstance(t, false, true, policy)
	cfg := &config.Config{Loaded: &config.LoadedLabConfig{}}
	require.NoError(t, inst.SyncPolicies(context.Background(), cfg))
	err := inst.client.Get(context.Background(), client.ObjectKeyFromObject(policy), &v1alpha1.ExporterAccessPolicy{})
	require.True(t, apierrors.IsNotFound(err), "unexpected Get result: %v", err)

	inst.config.Spec.Namespace = ""
	require.ErrorContains(t, inst.SyncPolicies(context.Background(), cfg), "namespace is required")
}
