package instance

import (
	"context"
	"errors"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
			cfg := &config.Config{Sources: config.Sources{Policies: []string{"policies/*.yaml"}}, Loaded: &config.LoadedLabConfig{
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
	cfg := &config.Config{Sources: config.Sources{Policies: []string{}}, Loaded: &config.LoadedLabConfig{}}
	require.NoError(t, inst.SyncPolicies(context.Background(), cfg))
	err := inst.client.Get(context.Background(), client.ObjectKeyFromObject(policy), &v1alpha1.ExporterAccessPolicy{})
	require.True(t, apierrors.IsNotFound(err), "unexpected Get result: %v", err)

	inst.config.Spec.Namespace = ""
	require.ErrorContains(t, inst.SyncPolicies(context.Background(), cfg), "namespace is required")
}

func TestSyncPoliciesContinuesAfterErrors(t *testing.T) {
	ctx := context.Background()
	inst := newPruneTestInstance(t, false, true)
	existing := make([]client.Object, 0, 4)
	for _, name := range []string{"b", "old-a", "old-b", "old-c"} {
		existing = append(existing, &v1alpha1.ExporterAccessPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: pruneTestNamespace},
		})
	}
	firstErr, secondErr := errors.New("first API failure"), errors.New("second API failure")
	failSync := true
	var attempted []string
	inst.client = fake.NewClientBuilder().WithScheme(inst.client.Scheme()).WithObjects(existing...).WithInterceptorFuncs(interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			attempted = append(attempted, "create "+obj.GetName())
			if failSync && obj.GetName() == "a" {
				return firstErr
			}
			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			attempted = append(attempted, "update "+obj.GetName())
			if failSync && obj.GetName() == "b" {
				return secondErr
			}
			return c.Update(ctx, obj, opts...)
		},
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			attempted = append(attempted, "delete "+obj.GetName())
			switch obj.GetName() {
			case "old-a":
				return firstErr
			case "old-b":
				return secondErr
			}
			return c.Delete(ctx, obj, opts...)
		},
	}).Build()
	cfg := &config.Config{
		Sources: config.Sources{Policies: []string{"policies/*.yaml"}},
		Loaded:  &config.LoadedLabConfig{Policies: map[string]*v1alpha1.ExporterAccessPolicy{}},
	}
	for _, name := range []string{"c", "b", "a"} {
		cfg.Loaded.Policies[name] = &v1alpha1.ExporterAccessPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}

	err := inst.SyncPolicies(ctx, cfg)
	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	assert.Equal(t, []string{"create a", "update b", "create c"}, attempted, "attempt all syncs in order, but skip pruning on failure")
	require.NoError(t, inst.client.Get(ctx, client.ObjectKey{Name: "c", Namespace: pruneTestNamespace}, &v1alpha1.ExporterAccessPolicy{}))

	failSync = false
	attempted = nil
	err = inst.SyncPolicies(ctx, cfg)
	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	assert.Equal(t, []string{"create a", "update b", "delete old-a", "delete old-b", "delete old-c"}, attempted)
	err = inst.client.Get(ctx, client.ObjectKey{Name: "old-c", Namespace: pruneTestNamespace}, &v1alpha1.ExporterAccessPolicy{})
	require.True(t, apierrors.IsNotFound(err), "later deletions must succeed despite earlier failures: %v", err)
}
