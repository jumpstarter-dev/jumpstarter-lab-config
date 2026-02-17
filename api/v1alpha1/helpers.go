package v1alpha1

const (
	DeadAnnotation       = "jumpstarter.dev/dead"
	LegacyDeadAnnotation = "dead"
	UnmanagedAnnotation  = "jumpstarter.dev/unmanaged"
)

func (e *ExporterInstance) HasConfigTemplate() bool {
	return e.Spec.ConfigTemplateRef.Name != ""
}

func (e *ExporterInstance) IsUnmanaged() (bool, string) {
	if e == nil || e.Annotations == nil {
		return false, ""
	}

	value, exists := e.Annotations[UnmanagedAnnotation]
	if !exists {
		return false, ""
	}

	return true, value
}

func (e *ExporterInstance) IsDead() (bool, string) {
	if e == nil || e.Annotations == nil {
		return false, ""
	}

	if value, exists := e.Annotations[DeadAnnotation]; exists {
		return true, value
	}

	// Backward compatibility for existing configs still using the legacy key.
	if value, exists := e.Annotations[LegacyDeadAnnotation]; exists {
		return true, value
	}

	return false, ""
}
