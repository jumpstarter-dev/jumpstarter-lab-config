package main

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	js "github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRotateExporterBatch(t *testing.T) {
	for _, scenario := range []string{
		"skip", "dry run wait", "force", "dry run force", "wait", "timeout", "missing lease", "canceled",
		"legacy wait", "legacy force", "legacy dry run",
	} {
		t.Run(scenario, func(t *testing.T) {
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(js.GroupVersion, &js.Exporter{})
			require.NoError(t, corev1.AddToScheme(scheme))
			// Keep new Lease fields that the pinned typed API does not know.
			scheme.AddKnownTypeWithName(js.GroupVersion.WithKind("Lease"), &unstructured.Unstructured{})
			var objects []client.Object
			for _, name := range []string{"busy-a", "free", "busy-b"} {
				exporter := &js.Exporter{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "lab"}}
				if name != "free" {
					exporter.Status.LeaseRef = &corev1.LocalObjectReference{Name: name + "-lease"}
					lease := &unstructured.Unstructured{Object: map[string]any{
						"apiVersion": js.GroupVersion.String(), "kind": "Lease",
						"metadata": map[string]any{"name": name + "-lease", "namespace": "lab"},
						"spec": map[string]any{"clientRef": map[string]any{"name": "alice"},
							"sharedWith": []any{"bob"}, "duration": "1h"},
						"status": map[string]any{"beginTime": "2026-11-24T10:00:00Z"},
					}}
					if scenario != "missing lease" || name != "busy-a" {
						objects = append(objects, lease)
					}
				}
				objects = append(objects, exporter)
				if strings.HasPrefix(scenario, "legacy") {
					exporter.UID = "exporter-uid"
					exporter.Status.Credential = &corev1.LocalObjectReference{Name: name + "-credential"}
					secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
						Name: name + "-credential", Namespace: "lab",
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(exporter,
							js.GroupVersion.WithKind("Exporter"))},
					}}
					if name != "busy-b" {
						secret.Annotations = map[string]string{"jumpstarter.dev/legacy-token-hash": "hash"}
					}
					objects = append(objects, secret)
				}
			}
			reads := map[string]int{}
			kc := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&js.Exporter{}).WithObjects(objects...).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey,
						obj client.Object, opts ...client.GetOption) error {
						if err := c.Get(ctx, key, obj, opts...); err != nil {
							return err
						}
						if exporter, ok := obj.(*js.Exporter); ok {
							reads[key.Name]++
							if scenario == "wait" && (key.Name == "busy-a" && reads[key.Name] == 2 ||
								key.Name == "busy-b" && reads[key.Name] == 3) {
								exporter.Status.LeaseRef = nil
								return c.Status().Update(ctx, exporter)
							}
						} else if secret, ok := obj.(*corev1.Secret); ok && scenario == "legacy wait" && reads["busy-a"] == 2 {
							// Another rotator migrates the credential while its lease remains active.
							secret.Annotations = nil
						} else if lease, ok := obj.(*unstructured.Unstructured); ok && scenario == "wait" && reads["busy-b"] == 2 {
							// A fresh API read must report a lease extension, not a cached end time.
							return unstructured.SetNestedField(lease.Object, "2026-11-24T12:00:00Z", "spec", "endTime")
						}
						return nil
					},
				}).Build()
			opts := rotationOptions{waitInterval: time.Millisecond, waitTimeout: 30 * time.Millisecond}
			opts.legacyOnly = strings.HasPrefix(scenario, "legacy")
			opts.wait = slices.Contains([]string{"wait", "timeout", "dry run wait", "legacy wait"}, scenario)
			switch scenario {
			case "force", "dry run force":
				opts.forceLeased = regexp.MustCompile("^busy-a$")
			case "legacy force", "legacy dry run":
				opts.forceLeased = regexp.MustCompile(".*")
			}
			opts.dryRun = strings.Contains(scenario, "dry run")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "canceled" {
				cancel()
			}
			var out bytes.Buffer
			var rotated []string
			err := rotateExporterBatch(ctx, kc, "lab", []string{"busy-a", "free", "busy-b"}, opts, &out,
				func(_ context.Context, name string, force bool) error {
					if force {
						require.Equal(t, "busy-a", name)
						require.Contains(t, out.String(), "alice, bob", "holders must be printed before forcing")
					}
					rotated = append(rotated, name)
					return nil
				})
			switch scenario {
			case "timeout":
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Contains(t, err.Error(), "busy-a, busy-b")
			case "canceled":
				require.ErrorIs(t, err, context.Canceled)
				require.Empty(t, rotated)
				return
			case "missing lease":
				require.ErrorContains(t, err, "read assigned lease")
			default:
				require.NoError(t, err)
			}
			switch scenario {
			case "wait":
				require.Equal(t, []string{"free", "busy-a", "busy-b"}, rotated)
				require.Contains(t, out.String(), "2026-11-24T12:00:00Z")
			case "force", "dry run force", "legacy force", "legacy dry run":
				require.Equal(t, []string{"busy-a", "free"}, rotated)
			default:
				require.Equal(t, []string{"free"}, rotated)
			}
			require.Contains(t, out.String(), "alice, bob")
			require.Contains(t, out.String(), "2026-11-24T11:00:00Z")
			require.Equal(t, 1, reads["free"], "do not rotate already processed exporters again")
			require.Equal(t, opts.legacyOnly,
				strings.Contains(out.String(), "Skipping exporter lab/busy-b: no legacy-token-hash credential"))
		})
	}
}
