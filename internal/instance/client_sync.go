package instance

import (
	"context"
	"fmt"

	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (i *Instance) SyncClients(ctx context.Context, cfg *config.Config) error {
	fmt.Printf("🔄 [%s] Syncing clients ===========================\n", i.config.Name)
	instanceClients, err := i.listClients(ctx)
	if err != nil {
		return fmt.Errorf("[%s] failed to list clients: %w", i.config.Name, err)
	}

	configClientMap := cfg.Loaded.Clients

	// create a clientMap from instanceClients
	instanceClientMap := make(map[string]v1alpha1.Client)
	for _, instClient := range instanceClients.Items {
		instanceClientMap[instClient.Name] = instClient
	}

	// delete clients that are not in config
	for _, instanceClient := range instanceClients.Items {
		if _, ok := configClientMap[instanceClient.Name]; !ok {
			err := i.deleteClient(ctx, instanceClient.Name)
			if err != nil {
				return fmt.Errorf("[%s] failed to delete client %s: %w", i.config.Name, instanceClient.Name, err)
			}
		}
	}

	// create clients that are in config but not in instance
	for _, cfgClient := range configClientMap {
		if _, ok := instanceClientMap[cfgClient.Name]; !ok {
			err := i.createClient(ctx, cfgClient)
			if err != nil {
				return fmt.Errorf("[%s] failed to create client %s: %w", i.config.Name, cfgClient.Name, err)
			}
		}
	}

	// update clients that are in both config and instance
	for _, instanceClient := range instanceClients.Items {
		if cfgClient, ok := configClientMap[instanceClient.Name]; ok {
			err := i.updateClient(ctx, &instanceClient, cfgClient)
			if err != nil {
				return fmt.Errorf("[%s] failed to update client %s: %w", i.config.Name, instanceClient.Name, err)
			}
		}
	}

	return nil
}

// listClients lists all clients in the instance's namespace
func (i *Instance) listClients(ctx context.Context) (*v1alpha1.ClientList, error) {
	clients := &v1alpha1.ClientList{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		// If no namespace specified, list from all namespaces
		err := i.client.List(ctx, clients)
		return clients, err
	}

	err := i.client.List(ctx, clients, client.InNamespace(namespace))
	return clients, err
}

// getClientByName retrieves a specific client by name
func (i *Instance) getClientByName(ctx context.Context, name string) (*v1alpha1.Client, error) {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required to get client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	return clientObj, err
}

// updateClient updates a client
func (i *Instance) updateClient(ctx context.Context, oldClientObj, clientObj *v1alpha1.Client) error {
	// Create a copy of the old object to preserve ResourceVersion and other metadata
	updatedClient := oldClientObj.DeepCopy()

	// Update the spec and other fields from the new config
	updatedClient.Spec = clientObj.Spec
	updatedClient.Labels = clientObj.Labels

	// Prepare metadata (annotations, namespace, etc.)
	// For updates, we want to preserve existing annotations and merge new ones
	i.prepareMetadata(&updatedClient.ObjectMeta, clientObj.Annotations)
	i.printDiff(oldClientObj, updatedClient, "client", updatedClient.Name)
	if i.dryRun {
		return nil
	}

	return i.client.Update(ctx, updatedClient)
}

// createClient creates a new client
func (i *Instance) createClient(ctx context.Context, clientObj *v1alpha1.Client) error {
	// Prepare metadata (annotations, namespace, etc.)
	i.prepareMetadata(&clientObj.ObjectMeta, clientObj.Annotations)

	if i.dryRun {
		fmt.Printf("➕ [%s] dry run: Would create client %s in namespace %s\n", i.config.Name, clientObj.Name, clientObj.Namespace)
		return nil
	} else {
		fmt.Printf("➕ [%s] Creating client %s in namespace %s\n", i.config.Name, clientObj.Name, clientObj.Namespace)
	}

	return i.client.Create(ctx, clientObj)
}

// deleteClient deletes a client by name
func (i *Instance) deleteClient(ctx context.Context, name string) error {
	clientObj := &v1alpha1.Client{}
	namespace := i.config.Spec.Namespace
	if namespace == "" {
		return fmt.Errorf("namespace is required to delete client %s", name)
	}

	err := i.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, clientObj)
	if err != nil {
		return fmt.Errorf("failed to get client %s: %w", name, err)
	}

	if i.dryRun {
		fmt.Printf("🗑️ [%s] dry run: Would delete client %s in namespace %s\n", i.config.Name, name, namespace)
		return nil
	}

	return i.client.Delete(ctx, clientObj)
}
