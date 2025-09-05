package config_lint

import (
	"fmt"
	"os"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
)

// Validate checks the loaded configuration for errors and prints a summary.
// It validates cross-references between objects and reports any issues found.
// If any errors are found, it prints them and exits the program with a non-zero status.
// If no errors are found, it prints the total number of variables and a success message.
func Validate(cfg *config.Config) {
	errorsByFile := Lint(cfg)
	if len(errorsByFile) > 0 {
		reportAllErrors(errorsByFile)
		os.Exit(1)
	}
	keys := cfg.Loaded.GetVariables().GetAllKeys()
	fmt.Printf("📚 Total Variables: %d\n", len(keys))
	fmt.Println("")
	fmt.Println("✅ All configurations are valid")
}

// ValidateWithError checks the loaded configuration for errors and returns an error if any are found.
// This version does not call os.Exit() and is suitable for use with profiling.
func ValidateWithError(cfg *config.Config) error {
	errorsByFile := Lint(cfg)
	if len(errorsByFile) > 0 {
		reportAllErrors(errorsByFile)
		return fmt.Errorf("validation failed with %d error(s)", len(errorsByFile))
	}
	keys := cfg.Loaded.GetVariables().GetAllKeys()
	fmt.Printf("📚 Total Variables: %d\n", len(keys))
	fmt.Println("")
	fmt.Println("✅ All configurations are valid")
	return nil
}

func Lint(cfg *config.Config) map[string][]error {
	// This function is a placeholder for the linting logic.
	// Currently, it only validates cross-references between objects.
	// The actual linting logic can be implemented here as needed.
	referencesErrors := validateReferences(cfg)
	templateErrors := validateTemplates(cfg)
	return mergeErrors(referencesErrors, templateErrors)
}

func mergeErrors(map1, map2 map[string][]error) map[string][]error {
	// create a new output map
	output := make(map[string][]error)

	// the output contains all the errors from map1 and map2,
	// if a key exists in both maps, the errors are appended

	for key, value := range map1 {
		output[key] = value
	}

	for key, value := range map2 {
		if _, exists := output[key]; !exists {
			output[key] = value
		} else {
			output[key] = append(output[key], value...)
		}
	}

	return output
}

// validateTemplates expands the templates and checks that the rendered templates are valid
func validateTemplates(cfg *config.Config) map[string][]error {
	errorsByItem := make(map[string][]error)

	// TODO: Implement template validation logic
	// This is a placeholder for template validation functionality
	for _, exporterInstance := range cfg.Loaded.GetExporterInstances() {
		// Template validation logic would go here
		// For now, this is just a placeholder that doesn't report any errors
		if exporterInstance.HasConfigTemplate() {
			errName := "ExporterInstance:" + exporterInstance.Name
			et, err := template.NewExporterInstanceTemplater(cfg, exporterInstance)
			if err != nil {
				errorsByItem[exporterInstance.Name] = append(errorsByItem[errName], err)
				continue
			}

			_, err = et.RenderTemplateLabels()
			if err != nil {
				errorsByItem[exporterInstance.Name] = append(errorsByItem[errName+" (labels)"], err)
			}
			_, err = et.RenderTemplateConfig()
			if err != nil {
				errorsByItem[exporterInstance.Name] = append(errorsByItem[errName+" (config)"], err)
			}
		}
	}

	return errorsByItem
}

// validateReferences checks that all cross-references between objects are valid
func validateReferences(cfg *config.Config) map[string][]error {
	errorsByFile := make(map[string][]error)

	// Helper function to get source file for an object
	getSourceFile := func(objectType, objectName string) string {
		if typeMap, exists := cfg.Loaded.GetSourceFiles()[objectType]; exists {
			if sourceFile, exists := typeMap[objectName]; exists {
				return sourceFile
			}
		}
		return "unknown"
	}

	// Helper function to add error to the map
	addError := func(sourceFile, errorMsg string) {
		if errorsByFile[sourceFile] == nil {
			errorsByFile[sourceFile] = make([]error, 0)
		}
		errorsByFile[sourceFile] = append(errorsByFile[sourceFile], fmt.Errorf("%s", errorMsg))
	}

	// Validate ExporterHost LocationRef references
	for name, host := range cfg.Loaded.GetExporterHosts() {
		if host.Spec.LocationRef.Name != "" {
			if _, exists := cfg.Loaded.GetPhysicalLocations()[host.Spec.LocationRef.Name]; !exists {
				sourceFile := getSourceFile("ExporterHost", name)
				addError(sourceFile, fmt.Sprintf("ExporterHost %s references non-existent location %s",
					name, host.Spec.LocationRef.Name))
			}
		}
	}

	// Validate ExporterInstance references
	for name, instance := range cfg.Loaded.GetExporterInstances() {
		sourceFile := getSourceFile("ExporterInstance", name)

		// Check DutLocationRef
		if instance.Spec.DutLocationRef.Name != "" {
			if _, exists := cfg.Loaded.GetPhysicalLocations()[instance.Spec.DutLocationRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent DUT location %s",
					name, instance.Spec.DutLocationRef.Name))
			}
		}

		// Check ExporterHostRef
		if instance.Spec.ExporterHostRef.Name != "" {
			if _, exists := cfg.Loaded.GetExporterHosts()[instance.Spec.ExporterHostRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent exporter host %s",
					name, instance.Spec.ExporterHostRef.Name))
			}
		}

		// Check JumpstarterInstanceRef
		if instance.Spec.JumpstarterInstanceRef.Name != "" {
			if _, exists := cfg.Loaded.GetJumpstarterInstances()[instance.Spec.JumpstarterInstanceRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent jumpstarter instance %s",
					name, instance.Spec.JumpstarterInstanceRef.Name))
			}
		}

		// Check ConfigTemplateRef
		if instance.Spec.ConfigTemplateRef.Name != "" {
			if _, exists := cfg.Loaded.GetExporterConfigTemplates()[instance.Spec.ConfigTemplateRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent config template %s",
					name, instance.Spec.ConfigTemplateRef.Name))
			}
		}
	}

	return errorsByFile
}

func reportAllErrors(errorsByFile map[string][]error) {
	totalErrors := 0
	for _, errors := range errorsByFile {
		totalErrors += len(errors)
	}
	fmt.Printf("\n❌ Validation failed with %d error(s):\n\n", totalErrors)

	for filename, errors := range errorsByFile {
		fmt.Printf("📄 %s:\n", filename)
		for _, err := range errors {
			fmt.Printf("\t🔹 %s\n", err)
		}
		fmt.Println()
	}
}
