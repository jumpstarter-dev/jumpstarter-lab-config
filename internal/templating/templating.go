package templating

import (
	"fmt"
	"reflect"
	"regexp"

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
	return t.applyTemplates(reflect.ValueOf(obj))
}

func (t *TemplateApplier) applyTemplates(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return t.applyTemplates(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return t.applyTemplates(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			// Only process exported fields
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			if err := t.applyTemplates(v.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := t.applyTemplates(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			// For map[string]string, apply directly
			if val.Kind() == reflect.String {
				str, err := ProcessTemplate(val.String(), t.variables, t.parameters)
				if err != nil {
					return fmt.Errorf("template error for map key %v: %w", key, err)
				}
				v.SetMapIndex(key, reflect.ValueOf(str))
			} else {
				// For other map value types, recurse
				if err := t.applyTemplates(val); err != nil {
					return err
				}
			}
		}
	case reflect.String:
		if v.CanSet() {
			str, err := ProcessTemplate(v.String(), t.variables, t.parameters)
			if err != nil {
				return err
			}
			v.SetString(str)
		}
	}
	return nil
}

func ProcessTemplate(data string, variables *vars.Variables, parameters *Parameters) (string, error) {
	// This function would process the template using the provided variables.
	// For now, we will just return the template as-is for demonstration purposes.
	// In a real implementation, you would use a templating engine like text/template or html/template.
	if needsReplacements(data) {
		replacements, err := constructReplacementMap(variables, parameters)
		if err != nil {
			return "", err
		}
		return applyReplacements(data, replacements)
	}

	return data, nil
}

func needsReplacements(data string) bool {
	// if "$(.*)" is found anywhere in the data, it indicates that replacements are needed
	// check with a regex
	return regexp.MustCompile(`\$\((.*?)\)`).MatchString(data)

}

func constructReplacementMap(variables *vars.Variables, parameters *Parameters) (map[string]string, error) {
	replacements := make(map[string]string)

	// Add variables to the replacement map
	for _, key := range variables.GetAllKeys() {
		value, err := variables.Get(key)
		if err != nil {
			return nil, fmt.Errorf("templating: error retrieving variable '%s': %w", key, err)
		}
		replacements["var."+key] = value
	}

	// Add parameters to the replacement map
	for key, value := range parameters.parameters {
		replacements["param."+key] = value
	}

	return replacements, nil
}

func applyReplacements(data string, replacements map[string]string) (string, error) {
	const RECURSION_LIMIT = 10 // Limit iterations to prevent infinite loops, i.e. var.a = $(var.a) or similar
	var (
		i                               int
		replacementsContainReplacements bool
		recursiveReplacementInfo        string
	)

	for i = 0; i < RECURSION_LIMIT; i++ {
		// DEBUG: fmt.Printf("Applying replacements, iteration %d, input: %s\n", i, data)
		replacementsContainReplacements = false
		// Apply replacements to the data
		for key, value := range replacements {
			keyRegexp := regexp.MustCompile(`\$\(\s*` + key + `\s*\)`)
			hasKeyReplacement := keyRegexp.MatchString(data)
			data = keyRegexp.ReplaceAllString(data, value)
			// if the key was found in the data, check if the value contains any replacements
			if hasKeyReplacement && needsReplacements(value) {
				// in such case, our replacement contains new replacements and we will need to iterate again
				replacementsContainReplacements = true
				recursiveReplacementInfo = fmt.Sprintf("%s => %s", key, value)
			}
		}
		if !replacementsContainReplacements {
			break // No more replacements needed
		}
	}
	// DEBUG: fmt.Printf("Finished replacements, iteration %d, output: %s\n", i, data)
	if i == RECURSION_LIMIT && replacementsContainReplacements {
		return data, fmt.Errorf("templating: recursion limit reached while applying replacements, "+
			"check for circular references, like: %s", recursiveReplacementInfo)
	}
	// find unhandled variables and return an error for the ones not found
	unhandled := regexp.MustCompile(`\$\(\s*(.*?)\s*\)`)
	matches := unhandled.FindAllStringSubmatch(data, -1)
	if len(matches) > 0 {
		var missingKeys []string
		for _, match := range matches {
			if len(match) > 1 {
				missingKeys = append(missingKeys, match[1])
			}
		}
		if len(missingKeys) > 0 {
			return data, fmt.Errorf("templating: unhandled variables found: %v", missingKeys)
		}
	}
	// If no unhandled variables are found, return the modified data
	return data, nil
}
