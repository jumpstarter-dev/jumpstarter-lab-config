package instance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alphaConfig "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
)

const pruneTestNamespace = "lab"

func newPruneTestInstance(t *testing.T, dryRun, prune bool, objs ...client.Object) *Instance {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	return &Instance{
		client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		config: &v1alphaConfig.JumpstarterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       v1alphaConfig.JumpstarterInstanceSpec{Namespace: pruneTestNamespace},
		},
		dryRun: dryRun,
		prune:  prune,
	}
}

func TestDeleteHonorsPrune(t *testing.T) {
	tests := []struct {
		name        string
		dryRun      bool
		prune       bool
		wantDeleted bool
	}{
		{name: "no prune keeps the object", dryRun: false, prune: false, wantDeleted: false},
		{name: "prune deletes the object", dryRun: false, prune: true, wantDeleted: true},
		{name: "dry-run with prune keeps the object", dryRun: true, prune: true, wantDeleted: false},
		{name: "dry-run without prune keeps the object", dryRun: true, prune: false, wantDeleted: false},
	}

	for _, tt := range tests {
		t.Run("exporter: "+tt.name, func(t *testing.T) {
			obj := &v1alpha1.Exporter{ObjectMeta: metav1.ObjectMeta{Name: "stale", Namespace: pruneTestNamespace}}
			inst := newPruneTestInstance(t, tt.dryRun, tt.prune, obj)

			require.NoError(t, inst.deleteExporter(context.Background(), "stale"))

			err := inst.client.Get(context.Background(), client.ObjectKeyFromObject(obj), &v1alpha1.Exporter{})
			assert.Equal(t, tt.wantDeleted, apierrors.IsNotFound(err), "unexpected Get result: %v", err)
		})

		t.Run("client: "+tt.name, func(t *testing.T) {
			obj := &v1alpha1.Client{ObjectMeta: metav1.ObjectMeta{Name: "stale", Namespace: pruneTestNamespace}}
			inst := newPruneTestInstance(t, tt.dryRun, tt.prune, obj)

			require.NoError(t, inst.deleteClient(context.Background(), "stale"))

			err := inst.client.Get(context.Background(), client.ObjectKeyFromObject(obj), &v1alpha1.Client{})
			assert.Equal(t, tt.wantDeleted, apierrors.IsNotFound(err), "unexpected Get result: %v", err)
		})
	}
}
