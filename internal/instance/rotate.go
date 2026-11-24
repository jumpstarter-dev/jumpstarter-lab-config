package instance

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RotateCredentials asks the controller to regenerate an existing credential.
// The caller must check delivery access before calling this for an exporter.
func (i *Instance) RotateCredentials(ctx context.Context, kind, name string) (*corev1.Secret, error) {
	var owner client.Object
	var credential *corev1.LocalObjectReference
	switch kind {
	case "exporter":
		obj, err := i.getExporterByName(ctx, name)
		if err != nil {
			return nil, err
		}
		owner, credential = obj, obj.Status.Credential
	case "client":
		obj, err := i.getClientByName(ctx, name)
		if err != nil {
			return nil, err
		}
		owner, credential = obj, obj.Status.Credential
	default:
		return nil, fmt.Errorf("unsupported credential kind %q", kind)
	}
	if credential == nil || credential.Name == "" {
		return nil, fmt.Errorf("%s %s has no internal credential Secret", kind, name)
	}
	key := client.ObjectKey{Namespace: owner.GetNamespace(), Name: credential.Name}
	old := &corev1.Secret{}
	if err := i.client.Get(ctx, key, old); err != nil {
		return nil, err
	}
	if !metav1.IsControlledBy(old, owner) || owner.GetUID() == "" {
		return nil, fmt.Errorf("credential Secret %s is not controlled by %s %s", key, kind, name)
	}
	if i.dryRun {
		return old, nil
	}
	// Avoid deleting a replacement created by a concurrent rotation.
	if err := i.client.Delete(ctx, old, client.Preconditions{
		UID: &old.UID, ResourceVersion: &old.ResourceVersion,
	}); err != nil {
		return nil, fmt.Errorf("delete credential Secret %s: %w", key, err)
	}
	replacement := &corev1.Secret{}
	err := wait.PollUntilContextTimeout(ctx, time.Second, 90*time.Second, true, func(ctx context.Context) (bool, error) {
		if err := i.client.Get(ctx, key, replacement); err != nil {
			return false, client.IgnoreNotFound(err)
		}
		if replacement.UID == old.UID || len(replacement.Data["token"]) == 0 {
			return false, nil
		}
		if !metav1.IsControlledBy(replacement, owner) {
			return false, fmt.Errorf("replacement Secret %s belongs to a different resource", key)
		}
		// Do not deliver a credential if its resource was recreated while waiting.
		current := owner.DeepCopyObject().(client.Object)
		if err := i.client.Get(ctx, client.ObjectKeyFromObject(owner), current); err != nil {
			return false, err
		}
		if current.GetUID() != owner.GetUID() {
			return false, fmt.Errorf("%s %s was recreated during rotation", kind, name)
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("credential Secret %s was deleted; waiting for controller regeneration failed: %w", key, err)
	}
	return replacement, nil
}
