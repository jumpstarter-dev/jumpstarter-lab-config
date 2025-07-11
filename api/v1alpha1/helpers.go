package v1alpha1

func (e *ExporterInstance) HasConfigTemplate() bool {
	return e.Spec.ConfigTemplateRef.Name != ""
}
