package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestResourceSyncEnabled(t *testing.T) {
	for _, tt := range []struct {
		name     string
		cfg      ResourceSyncConfig
		global   []string
		fallback *bool
		want     bool
	}{
		{name: "no sources skips sync"},
		{name: "legacy true without sources skips sync", fallback: ptr.To(true)},
		{name: "global sources enable sync", global: []string{"clients/*.yaml"}, want: true},
		{name: "empty global sources enable sync", global: []string{}, want: true},
		{name: "inherit disabled", global: []string{"clients/*.yaml"}, fallback: ptr.To(false)},
		{name: "inherit enabled", global: []string{"clients/*.yaml"}, fallback: ptr.To(true), want: true},
		{name: "sources enable sync", cfg: ResourceSyncConfig{Sources: []string{"clients/*.yaml"}}, fallback: ptr.To(false), want: true},
		{name: "explicit false wins", cfg: ResourceSyncConfig{Enabled: ptr.To(false), Sources: []string{"clients/*.yaml"}}, fallback: ptr.To(true)},
		{name: "explicit true wins", cfg: ResourceSyncConfig{Enabled: ptr.To(true)}, fallback: ptr.To(false), want: true},
		{name: "empty instance sources enable sync", cfg: ResourceSyncConfig{Sources: []string{}}, fallback: ptr.To(false), want: true},
		{name: "explicit false overrides global sources", cfg: ResourceSyncConfig{Enabled: ptr.To(false)}, global: []string{"clients/*.yaml"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.IsEnabled(tt.global, tt.fallback))
		})
	}
}

func writeInstanceConfig(t *testing.T, overrides string) string {
	t.Helper()
	dir := t.TempDir()
	for _, resource := range []struct{ file, kind, name, label string }{
		{"global-client", "Client", "global-client", "global"},
		{"production-client", "Client", "scoped-client", "production"},
		{"staging-client", "Client", "scoped-client", "staging"},
		{"global-policy", "ExporterAccessPolicy", "global-policy", "global"},
		{"production-policy", "ExporterAccessPolicy", "scoped-policy", "production"},
		{"staging-policy", "ExporterAccessPolicy", "scoped-policy", "staging"},
	} {
		content := fmt.Sprintf("apiVersion: jumpstarter.dev/v1alpha1\nkind: %s\nmetadata:\n  name: %s\n  labels:\n    source: %s\n",
			resource.kind, resource.name, resource.label)
		require.NoError(t, os.WriteFile(filepath.Join(dir, resource.file+".yaml"), []byte(content), 0600))
	}
	for _, name := range []string{"production", "staging", "empty", "inherited"} {
		content := fmt.Sprintf("apiVersion: meta.jumpstarter.dev/v1alpha1\nkind: JumpstarterInstance\nmetadata:\n  name: %s\nspec:\n  namespace: %s\n", name, name)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "instance-"+name+".yaml"), []byte(content), 0600))
	}
	content := `sources:
  clients: [global-client.yaml]
  policies: [global-policy.yaml]
  jumpstarter_instances: [instance-*.yaml]
` + overrides
	path := filepath.Join(dir, "lab.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestInstanceSources(t *testing.T) {
	path := writeInstanceConfig(t, `instances:
  production:
    clients:
      enabled: false
      sources: [production-client.yaml]
    policies:
      sources: [production-policy.yaml]
  staging:
    clients:
      sources: [global-client.yaml, staging-client.yaml]
    policies:
      sources: [global-policy.yaml, staging-policy.yaml]
  empty:
    clients:
      sources: []
    policies:
      sources: []
  inherited:
    clients:
      enabled: true
`)
	cfg, err := LoadConfig(path, "")
	require.NoError(t, err)
	loaded := cfg.Loaded
	assert.Len(t, loaded.Clients, 1)
	assert.Len(t, loaded.Policies, 1)
	assert.Len(t, loaded.GetClientsForInstance("production"), 1)
	assert.Len(t, loaded.GetPoliciesForInstance("production"), 1)
	assert.Len(t, loaded.GetClientsForInstance("staging"), 2)
	assert.Len(t, loaded.GetPoliciesForInstance("staging"), 2)
	assert.Equal(t, "production", loaded.GetClientsForInstance("production")["scoped-client"].Labels["source"])
	assert.Equal(t, "staging", loaded.GetClientsForInstance("staging")["scoped-client"].Labels["source"])
	assert.Equal(t, "production", loaded.GetPoliciesForInstance("production")["scoped-policy"].Labels["source"])
	assert.Equal(t, "staging", loaded.GetPoliciesForInstance("staging")["scoped-policy"].Labels["source"])
	assert.Empty(t, loaded.GetClientsForInstance("empty"))
	assert.Empty(t, loaded.GetPoliciesForInstance("empty"))
	assert.Equal(t, loaded.Clients, loaded.GetClientsForInstance("inherited"))
	assert.Equal(t, loaded.Policies, loaded.GetPoliciesForInstance("inherited"))
	assert.False(t, cfg.Instances["production"].Clients.IsEnabled(cfg.Sources.Clients, nil))
	assert.True(t, cfg.Instances["production"].Policies.IsEnabled(cfg.Sources.Policies, nil))

	// Configurations without the new section continue to use the global sets.
	legacy, err := LoadConfig(writeInstanceConfig(t, ""), "")
	require.NoError(t, err)
	assert.Equal(t, legacy.Loaded.Clients, legacy.Loaded.GetClientsForInstance("production"))
	assert.Equal(t, legacy.Loaded.Policies, legacy.Loaded.GetPoliciesForInstance("production"))
}

func TestInvalidInstanceSources(t *testing.T) {
	for _, tt := range []struct{ name, overrides, want string }{
		{"unknown instance", "instances:\n  typo: {}\n", "unknown JumpstarterInstance"},
		{"duplicate clients", "instances:\n  production:\n    clients:\n      sources: [production-client.yaml, staging-client.yaml]\n", "duplicate Client name"},
		{"duplicate policies", "instances:\n  production:\n    policies:\n      sources: [production-policy.yaml, staging-policy.yaml]\n", "duplicate ExporterAccessPolicy name"},
		{"wrong resource kind", "instances:\n  production:\n    policies:\n      sources: [production-client.yaml]\n", "instances.production.policies"},
		{"invalid glob", "instances:\n  production:\n    clients:\n      sources: ['[']\n", "instances.production.clients"},
		{"invalid enabled", "instances:\n  production:\n    clients:\n      enabled: sometimes\n", "cannot unmarshal"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(writeInstanceConfig(t, tt.overrides), "")
			require.ErrorContains(t, err, tt.want)
		})
	}
}
