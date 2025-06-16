package config

import (
	"fmt"
	"os"
)

// Validate checks the loaded configuration for errors and prints a summary.
// It validates cross-references between objects and reports any issues found.
// If any errors are found, it prints them and exits the program with a non-zero status.
// If no errors are found, it prints the total number of variables and a success message.
func (cfg *Config) Validate() {
	errorsByFile := cfg.Lint()
	if len(errorsByFile) > 0 {
		reportAllErrors(errorsByFile)
		os.Exit(1)
	}
	keys := cfg.Loaded.Variables.GetAllKeys()
	fmt.Printf("📚 Total Variables: %d\n", len(keys))
	fmt.Println("")
	fmt.Println("✅ All configurations are valid")
}

func (cfg *Config) Lint() map[string][]error {
	// This function is a placeholder for the linting logic.
	// Currently, it only validates cross-references between objects.
	// The actual linting logic can be implemented here as needed.
	return validateReferences(cfg.Loaded)
}

// validateReferences checks that all cross-references between objects are valid
func validateReferences(loaded *LoadedLabConfig) map[string][]error {
	errorsByFile := make(map[string][]error)

	// Helper function to get source file for an object
	getSourceFile := func(objectType, objectName string) string {
		if typeMap, exists := loaded.SourceFiles[objectType]; exists {
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
	for name, host := range loaded.ExporterHosts {
		if host.Spec.LocationRef.Name != "" {
			if _, exists := loaded.PhysicalLocations[host.Spec.LocationRef.Name]; !exists {
				sourceFile := getSourceFile("ExporterHost", name)
				addError(sourceFile, fmt.Sprintf("ExporterHost %s references non-existent location %s",
					name, host.Spec.LocationRef.Name))
			}
		}
	}

	// Validate ExporterInstance references
	for name, instance := range loaded.ExporterInstances {
		sourceFile := getSourceFile("ExporterInstance", name)

		// Check DutLocationRef
		if instance.Spec.DutLocationRef.Name != "" {
			if _, exists := loaded.PhysicalLocations[instance.Spec.DutLocationRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent DUT location %s",
					name, instance.Spec.DutLocationRef.Name))
			}
		}

		// Check ExporterHostRef
		if instance.Spec.ExporterHostRef.Name != "" {
			if _, exists := loaded.ExporterHosts[instance.Spec.ExporterHostRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent exporter host %s",
					name, instance.Spec.ExporterHostRef.Name))
			}
		}

		// Check JumpstarterInstanceRef
		if instance.Spec.JumpstarterInstanceRef.Name != "" {
			if _, exists := loaded.JumpstarterInstances[instance.Spec.JumpstarterInstanceRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent jumpstarter instance %s",
					name, instance.Spec.JumpstarterInstanceRef.Name))
			}
		}

		// Check ConfigTemplateRef
		if instance.Spec.ConfigTemplateRef.Name != "" {
			if _, exists := loaded.ExporterConfigTemplates[instance.Spec.ConfigTemplateRef.Name]; !exists {
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
