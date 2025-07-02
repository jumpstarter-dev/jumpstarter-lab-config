package v1alpha1

import (
	"github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
)

func (e *ExporterInstance) GetExporterObjectForInstance(jumpstarterInstance string) *v1alpha1.Exporter {
	// If this exporter instance is targeting the given jumpstarter instance, return the exporter object
	if e.Spec.JumpstarterInstanceRef.Name == jumpstarterInstance {
		return &v1alpha1.Exporter{
			TypeMeta:   e.TypeMeta,
			ObjectMeta: e.ObjectMeta,
			Spec: v1alpha1.ExporterSpec{
				Username: &e.Spec.Username,
			},
		}
	}
	return nil
}
