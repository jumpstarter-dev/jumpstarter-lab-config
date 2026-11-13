package instance

import (
	"context"
	"regexp"
	"testing"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDisabledInstanceSync(t *testing.T) {
	for _, enabled := range []*bool{nil, ptr.To(false), ptr.To(true)} {
		inst := newPruneTestInstance(t, false, true)
		inst.config.Spec.SyncClients = ptr.To(false)
		cfg := &config.Config{
			Instances: map[string]config.InstanceConfig{"test": {
				Clients:  config.ResourceSyncConfig{Enabled: enabled},
				Policies: config.ResourceSyncConfig{Enabled: ptr.To(false)},
			}},
			Loaded: &config.LoadedLabConfig{Clients: map[string]*v1alpha1.Client{
				"new": {ObjectMeta: metav1.ObjectMeta{Name: "new"}},
			}},
		}
		if enabled == nil || !*enabled {
			// Disabled sync must not even contact Kubernetes, including with --prune.
			inst.client = nil
		}
		require.NoError(t, inst.SyncClients(context.Background(), cfg, nil))
		if enabled != nil && *enabled {
			require.NoError(t, inst.client.Get(context.Background(),
				client.ObjectKey{Name: "new", Namespace: pruneTestNamespace}, &v1alpha1.Client{}))
		}
		inst.client = nil
		inst.config.Spec.Namespace = ""
		require.NoError(t, inst.SyncPolicies(context.Background(), cfg))
	}

	// An explicit false also overrides the default of syncing clients.
	inst := newPruneTestInstance(t, false, true)
	inst.client = nil
	cfg := &config.Config{Instances: map[string]config.InstanceConfig{"test": {
		Clients: config.ResourceSyncConfig{Enabled: ptr.To(false)},
	}}}
	require.NoError(t, inst.SyncClients(context.Background(), cfg, nil))
}

func TestInstanceResourceSets(t *testing.T) {
	ctx := context.Background()
	globalClient := &v1alpha1.Client{ObjectMeta: metav1.ObjectMeta{Name: "global", Namespace: "manifest"}}
	globalPolicy := &v1alpha1.ExporterAccessPolicy{ObjectMeta: metav1.ObjectMeta{Name: "global", Namespace: "manifest"}}
	selectedClient := &v1alpha1.Client{ObjectMeta: metav1.ObjectMeta{Name: "selected", Labels: map[string]string{"updated": "yes"}}}
	selectedPolicy := &v1alpha1.ExporterAccessPolicy{ObjectMeta: metav1.ObjectMeta{Name: "selected", Labels: map[string]string{"updated": "yes"}}}
	cfg := &config.Config{Loaded: &config.LoadedLabConfig{
		Clients:  map[string]*v1alpha1.Client{"global": globalClient},
		Policies: map[string]*v1alpha1.ExporterAccessPolicy{"global": globalPolicy},
		InstanceClients: map[string]map[string]*v1alpha1.Client{
			"scoped": {"selected": selectedClient}, "empty": {},
		},
		InstancePolicies: map[string]map[string]*v1alpha1.ExporterAccessPolicy{
			"scoped": {"selected": selectedPolicy}, "empty": {},
		},
	}}
	for _, mode := range []struct {
		name   string
		prune  bool
		dryRun bool
	}{
		{name: "apply"}, {name: "prune", prune: true}, {name: "dry run", prune: true, dryRun: true},
	} {
		t.Run(mode.name, func(t *testing.T) {
			inst := newPruneTestInstance(t, mode.dryRun, mode.prune)
			for _, name := range []string{"scoped", "inherited", "empty", "another-inherited"} {
				inst.config.Name = name
				inst.config.Spec.Namespace = name
				for _, resourceName := range []string{"selected", "stale"} {
					meta := metav1.ObjectMeta{Name: resourceName, Namespace: name}
					require.NoError(t, inst.client.Create(ctx, &v1alpha1.Client{ObjectMeta: meta}))
					require.NoError(t, inst.client.Create(ctx, &v1alpha1.ExporterAccessPolicy{ObjectMeta: meta}))
				}
				require.NoError(t, inst.SyncClients(ctx, cfg, nil))
				require.NoError(t, inst.SyncPolicies(ctx, cfg))
				clients := &v1alpha1.ClientList{}
				policies := &v1alpha1.ExporterAccessPolicyList{}
				require.NoError(t, inst.client.List(ctx, clients, client.InNamespace(name)))
				require.NoError(t, inst.client.List(ctx, policies, client.InNamespace(name)))
				want := map[string]map[string]string{}
				if !mode.prune || mode.dryRun {
					want["selected"], want["stale"] = nil, nil
				}
				if !mode.dryRun {
					for key, obj := range cfg.Loaded.GetClientsForInstance(name) {
						want[key] = obj.Labels
					}
				}
				gotClients, gotPolicies := map[string]map[string]string{}, map[string]map[string]string{}
				for _, obj := range clients.Items {
					gotClients[obj.Name] = obj.Labels
				}
				for _, obj := range policies.Items {
					gotPolicies[obj.Name] = obj.Labels
				}
				assert.Equal(t, want, gotClients, name)
				assert.Equal(t, want, gotPolicies, name)
			}
		})
	}
	assert.Equal(t, "manifest", globalClient.Namespace)
	assert.Empty(t, globalClient.ResourceVersion)
	assert.Empty(t, globalClient.Annotations)
	assert.Empty(t, selectedClient.Namespace)
	assert.Empty(t, selectedClient.Annotations)
	assert.Equal(t, "manifest", globalPolicy.Namespace)
}

func TestInstanceClientFilter(t *testing.T) {
	ctx := context.Background()
	kept := &v1alpha1.Client{ObjectMeta: metav1.ObjectMeta{Name: "unmatched", Namespace: pruneTestNamespace}}
	inst := newPruneTestInstance(t, false, true, kept)
	cfg := &config.Config{Loaded: &config.LoadedLabConfig{
		InstanceClients: map[string]map[string]*v1alpha1.Client{"test": {
			"selected": {ObjectMeta: metav1.ObjectMeta{Name: "selected"}},
			"excluded": {ObjectMeta: metav1.ObjectMeta{Name: "excluded"}},
		}},
	}}
	require.NoError(t, inst.SyncClients(ctx, cfg, regexp.MustCompile("^selected$")))
	clients := &v1alpha1.ClientList{}
	require.NoError(t, inst.client.List(ctx, clients))
	names := make([]string, 0, len(clients.Items))
	for _, obj := range clients.Items {
		names = append(names, obj.Name)
	}
	assert.ElementsMatch(t, []string{"selected", "unmatched"}, names)
}
