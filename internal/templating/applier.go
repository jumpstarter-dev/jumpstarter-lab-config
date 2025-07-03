package templating

import (
	"fmt"
	"reflect"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"
)

type TemplateApplier struct {
	variables  *vars.Variables
	parameters *Parameters
}

func NewTemplateApplier(cfg *config.Config, parameters *Parameters) (*TemplateApplier, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.Loaded == nil {
		return nil, fmt.Errorf("loaded config cannot be nil")
	}
	return &TemplateApplier{
		variables:  cfg.Loaded.Variables,
		parameters: parameters,
	}, nil
}

// ApplyTemplatesRecursively walks through all fields of the given object recursively,
// and applies ProcessTemplate to every string field.
func (t *TemplateApplier) Apply(obj interface{}) error {
	meta := createMetadataParameters(obj)
	return t.applyTemplates(reflect.ValueOf(obj), meta, nil)
}

func (t *TemplateApplier) ApplyWithParameters(obj interface{}, customParameters *Parameters) error {
	meta := createMetadataParameters(obj)
	return t.applyTemplates(reflect.ValueOf(obj), meta, customParameters)
}

func createMetadataParameters(obj interface{}) *Parameters {
	meta := NewParameters("meta")

	if obj != nil {
		val := reflect.ValueOf(obj)

		// If obj is a pointer, get the element it points to
		if val.Kind() == reflect.Ptr {
			if val.IsNil() { // Check if the pointer is nil
				obj = nil // Treat as nil object if pointer is nil
			} else {
				val = val.Elem()
			}
		}

		if obj != nil && val.Kind() == reflect.Struct {
			// Try to get obj.Name
			nameField := val.FieldByName("Name")
			if nameField.IsValid() && nameField.Kind() == reflect.String && nameField.CanInterface() {
				meta.Set("name", nameField.String())
			} else {
				// If obj.Name was not found or not a string, try obj.Metadata.Name
				metadataField := val.FieldByName("Metadata")
				if metadataField.IsValid() && metadataField.CanInterface() {
					metadataVal := metadataField
					// Get the actual value of Metadata, handling if it's a pointer
					if metadataVal.Kind() == reflect.Ptr {
						if metadataVal.IsNil() {
							return meta
						}
						metadataVal = metadataVal.Elem()
					}

					if metadataVal.Kind() == reflect.Struct {
						nameFromMetadataField := metadataVal.FieldByName("Name")
						if nameFromMetadataField.IsValid() && nameFromMetadataField.Kind() == reflect.String && nameFromMetadataField.CanInterface() {
							meta.Set("name", nameFromMetadataField.String())
						}
					}
				}
			}
		}
	}
	return meta
}

func (t *TemplateApplier) applyTemplates(v reflect.Value, meta *Parameters, customParameters *Parameters) error {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return t.applyTemplates(v.Elem(), meta, customParameters)
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return t.applyTemplates(v.Elem(), meta, customParameters)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			// Only process exported fields
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			if err := t.applyTemplates(v.Field(i), meta, customParameters); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := t.applyTemplates(v.Index(i), meta, customParameters); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			// For map[string]string, apply directly
			if val.Kind() == reflect.String {
				str, err := ProcessTemplate(val.String(), t.variables, t.parameters.Merge(customParameters), meta)
				if err != nil {
					return fmt.Errorf("template error for map key %v: %w", key, err)
				}
				v.SetMapIndex(key, reflect.ValueOf(str))
			} else {
				// For other map value types, recurse
				if err := t.applyTemplates(val, meta, customParameters); err != nil {
					return err
				}
			}
		}
	case reflect.String:
		if v.CanSet() {
			str, err := ProcessTemplate(v.String(), t.variables, t.parameters.Merge(customParameters), meta)
			if err != nil {
				return err
			}
			v.SetString(str)
		}
	}
	return nil
}
