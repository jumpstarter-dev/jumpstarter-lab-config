package instance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	js "github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRotateCredentials(t *testing.T) {
	for _, kind := range []string{"client", "exporter"} {
		for _, scenario := range []string{
			"rotate", "dry run", "wrong owner", "missing credential", "delete denied", "timeout", "stale secret", "recreated owner",
			"legacy rotate", "legacy dry run", "legacy missing hash", "legacy empty hash", "legacy no credential",
			"legacy migrated", "legacy changed before delete", "legacy wrong owner",
		} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				scheme := runtime.NewScheme()
				require.NoError(t, js.AddToScheme(scheme))
				require.NoError(t, corev1.AddToScheme(scheme))
				metadata := metav1.ObjectMeta{Name: "test", Namespace: "lab", UID: "resource-uid"}
				ref := &corev1.LocalObjectReference{Name: "credential"}
				if scenario == "missing credential" || scenario == "legacy no credential" {
					ref = nil
				}
				var owner client.Object = &js.Client{ObjectMeta: metadata, Status: js.ClientStatus{Credential: ref}}
				if kind == "exporter" {
					owner = &js.Exporter{ObjectMeta: metadata, Status: js.ExporterStatus{Credential: ref}}
				}
				controller := true
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "credential", Namespace: "lab", UID: "old-secret",
						OwnerReferences: []metav1.OwnerReference{{UID: owner.GetUID(), Controller: &controller}}},
					Data: map[string][]byte{"token": []byte("old-token")},
				}
				legacyOnly := strings.HasPrefix(scenario, "legacy ")
				if legacyOnly && scenario != "legacy missing hash" {
					secret.Annotations = map[string]string{"jumpstarter.dev/legacy-token-hash": "old-token-hash"}
				}
				if scenario == "legacy empty hash" {
					secret.Annotations["jumpstarter.dev/legacy-token-hash"] = ""
				}
				if scenario == "wrong owner" || scenario == "legacy wrong owner" {
					secret.OwnerReferences[0].UID = "someone-else"
				}
				deletes := 0
				kc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(owner, secret).
					WithInterceptorFuncs(interceptor.Funcs{
						Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
							deletes++
							options := (&client.DeleteOptions{}).ApplyOptions(opts)
							require.Equal(t, secret.UID, *options.Preconditions.UID)
							require.NotEmpty(t, *options.Preconditions.ResourceVersion)
							if scenario == "delete denied" {
								return errors.New("forbidden")
							}
							if scenario == "stale secret" {
								return nil // Asynchronous deletion: the old Secret is still visible.
							}
							if scenario == "legacy changed before delete" {
								current := &corev1.Secret{}
								require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(obj), current))
								current.Annotations = nil
								require.NoError(t, c.Update(ctx, current))
							}
							if err := c.Delete(ctx, obj, opts...); err != nil {
								return err
							}
							if scenario == "timeout" {
								return nil
							}
							if scenario == "recreated owner" {
								require.NoError(t, c.Delete(ctx, owner))
								copy := owner.DeepCopyObject().(client.Object)
								copy.SetUID("new-resource-uid")
								copy.SetResourceVersion("")
								require.NoError(t, c.Create(ctx, copy))
							}
							replacement := secret.DeepCopy()
							replacement.UID = "new-secret"
							replacement.ResourceVersion = ""
							replacement.Data["token"] = []byte("replacement-token")
							replacement.Annotations = nil
							return c.Create(ctx, replacement)
						},
					}).Build()
				inst := &Instance{client: kc, dryRun: strings.HasSuffix(scenario, "dry run"), config: &api.JumpstarterInstance{
					Spec: api.JumpstarterInstanceSpec{Namespace: "lab"},
				}}
				if scenario == "legacy migrated" {
					selected, err := CredentialForRotation(ctx, kc, owner, ref, true)
					require.NoError(t, err)
					selected.Annotations = nil
					require.NoError(t, kc.Update(ctx, selected))
				}
				got, err := inst.RotateCredentials(ctx, kind, "test", legacyOnly)
				switch scenario {
				case "rotate", "legacy rotate":
					require.NoError(t, err)
					require.Equal(t, "replacement-token", string(got.Data["token"]))
				case "dry run", "legacy dry run":
					require.NoError(t, err)
					require.Zero(t, deletes)
					require.Equal(t, "old-token", string(got.Data["token"]))
				case "legacy missing hash", "legacy empty hash", "legacy no credential", "legacy migrated":
					require.ErrorIs(t, err, ErrNotLegacyCredential)
					require.Nil(t, got)
					require.Zero(t, deletes)
				case "wrong owner", "missing credential", "legacy wrong owner":
					require.Error(t, err)
					require.Zero(t, deletes)
				default:
					require.Error(t, err)
					require.Equal(t, 1, deletes)
				}
				if scenario == "legacy changed before delete" {
					remaining := &corev1.Secret{}
					require.NoError(t, kc.Get(ctx, client.ObjectKeyFromObject(secret), remaining))
					require.Equal(t, secret.UID, remaining.UID)
				}
			})
		}
	}
}
