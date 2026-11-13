package instance

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncPolicies synchronizes access policies in the instance's namespace.
func (i *Instance) SyncPolicies(ctx context.Context, cfg *config.Config) error {
	if !cfg.Instances[i.config.Name].Policies.IsEnabled(cfg.Sources.Policies, nil) {
		fmt.Printf("\n⏸️  [%s] Policy sync disabled\n", i.config.Name)
		return nil
	}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("[%s] namespace is required to sync policies", i.config.Name)
	}

	fmt.Printf("\n🔄 [%s] Syncing policies ===========================\n\n", i.config.Name)
	policies := &v1alpha1.ExporterAccessPolicyList{}
	if err := i.client.List(ctx, policies, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("[%s] failed to list policies: %w", i.config.Name, err)
	}
	current := make(map[string]*v1alpha1.ExporterAccessPolicy, len(policies.Items))
	for idx := range policies.Items {
		policy := &policies.Items[idx]
		current[policy.Name] = policy
	}

	configPolicies := cfg.Loaded.GetPoliciesForInstance(i.config.Name)
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(configPolicies)) {
		if err := i.syncPolicy(ctx, current[name], configPolicies[name]); err != nil {
			errs = append(errs, fmt.Errorf("[%s] failed to sync policy %s: %w", i.config.Name, name, err))
		}
	}
	// Keep old access policies if their replacements could not be applied.
	if err := errors.Join(errs...); err != nil {
		return err
	}

	for _, name := range slices.Sorted(maps.Keys(current)) {
		if _, ok := configPolicies[name]; ok {
			continue
		}
		if err := i.deletePolicy(ctx, current[name]); err != nil {
			errs = append(errs, fmt.Errorf("[%s] failed to delete policy %s: %w", i.config.Name, name, err))
		}
	}
	return errors.Join(errs...)
}

func (i *Instance) syncPolicy(ctx context.Context, oldPolicy, policy *v1alpha1.ExporterAccessPolicy) error {
	// Each instance gets its own copy, including its namespace and server metadata.
	desired := policy.DeepCopy()
	desired.Namespace = i.config.Spec.Namespace
	if oldPolicy != nil {
		labels := desired.Labels
		desired.TypeMeta = oldPolicy.TypeMeta
		desired.ObjectMeta = *oldPolicy.ObjectMeta.DeepCopy()
		desired.Labels = labels
		desired.Status = oldPolicy.Status
	}
	i.prepareMetadata(&desired.ObjectMeta, policy.Annotations)

	if oldPolicy != nil {
		changed := i.checkAndPrintDiff(oldPolicy, desired, "policy", desired.Name, i.dryRun)
		if i.dryRun || !changed {
			return nil
		}
		return i.client.Update(ctx, desired)
	}

	if i.dryRun {
		fmt.Printf("➕ [%s] Would create policy %s in namespace %s\n", i.config.Name, desired.Name, desired.Namespace)
		return nil
	}
	fmt.Printf("➕ [%s] Creating policy %s in namespace %s\n", i.config.Name, desired.Name, desired.Namespace)
	return i.client.Create(ctx, desired)
}

func (i *Instance) deletePolicy(ctx context.Context, policy *v1alpha1.ExporterAccessPolicy) error {
	if !i.prune {
		fmt.Printf("⚠️ [%s] Policy %s in namespace %s is not in the configuration; use --prune to delete it\n",
			i.config.Name, policy.Name, policy.Namespace)
		return nil
	}
	if i.dryRun {
		fmt.Printf("🗑️ [%s] Would delete policy %s in namespace %s\n", i.config.Name, policy.Name, policy.Namespace)
		return nil
	}
	fmt.Printf("🗑️ [%s] Deleting policy %s in namespace %s\n", i.config.Name, policy.Name, policy.Namespace)
	return i.client.Delete(ctx, policy)
}
