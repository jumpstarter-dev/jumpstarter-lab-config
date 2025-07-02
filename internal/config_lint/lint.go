package config_lint

import (
	"fmt"
	"os"

	jsApi "github.com/jumpstarter-dev/jumpstarter-controller/api/v1alpha1"
	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/vars"
)

// LintableConfig defines the interface that any configuration must implement to be lintable.
// This avoids circular dependencies by not importing the full config package.
type LintableConfig interface {
	GetClients() map[string]*jsApi.Client
	GetPolicies() map[string]*jsApi.ExporterAccessPolicy
	GetPhysicalLocations() map[string]*api.PhysicalLocation
	GetExporterHosts() map[string]*api.ExporterHost
	GetExporterInstances() map[string]*api.ExporterInstance
	GetExporterConfigTemplates() map[string]*api.ExporterConfigTemplate
	GetJumpstarterInstances() map[string]*api.JumpstarterInstance
	GetVariables() *vars.Variables
	GetSourceFiles() map[string]map[string]string
}

// Validate checks the loaded configuration for errors and prints a summary.
// It validates cross-references between objects and reports any issues found.
// If any errors are found, it prints them and exits the program with a non-zero status.
// If no errors are found, it prints the total number of variables and a success message.
func Validate(config LintableConfig) {
	errorsByFile := Lint(config)
	if len(errorsByFile) > 0 {
		reportAllErrors(errorsByFile)
		os.Exit(1)
	}
	keys := config.GetVariables().GetAllKeys()
	fmt.Printf("📚 Total Variables: %d\n", len(keys))
	fmt.Println("")
	fmt.Println("✅ All configurations are valid")
}

func Lint(config LintableConfig) map[string][]error {
	// This function is a placeholder for the linting logic.
	// Currently, it only validates cross-references between objects.
	// The actual linting logic can be implemented here as needed.
	return validateReferences(config)
}

// validateReferences checks that all cross-references between objects are valid
func validateReferences(config LintableConfig) map[string][]error {
	errorsByFile := make(map[string][]error)

	// Helper function to get source file for an object
	getSourceFile := func(objectType, objectName string) string {
		if typeMap, exists := config.GetSourceFiles()[objectType]; exists {
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
	for name, host := range config.GetExporterHosts() {
		if host.Spec.LocationRef.Name != "" {
			if _, exists := config.GetPhysicalLocations()[host.Spec.LocationRef.Name]; !exists {
				sourceFile := getSourceFile("ExporterHost", name)
				addError(sourceFile, fmt.Sprintf("ExporterHost %s references non-existent location %s",
					name, host.Spec.LocationRef.Name))
			}
		}
	}

	// Validate ExporterInstance references
	for name, instance := range config.GetExporterInstances() {
		sourceFile := getSourceFile("ExporterInstance", name)

		// Check DutLocationRef
		if instance.Spec.DutLocationRef.Name != "" {
			if _, exists := config.GetPhysicalLocations()[instance.Spec.DutLocationRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent DUT location %s",
					name, instance.Spec.DutLocationRef.Name))
			}
		}

		// Check ExporterHostRef
		if instance.Spec.ExporterHostRef.Name != "" {
			if _, exists := config.GetExporterHosts()[instance.Spec.ExporterHostRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent exporter host %s",
					name, instance.Spec.ExporterHostRef.Name))
			}
		}

		// Check JumpstarterInstanceRef
		if instance.Spec.JumpstarterInstanceRef.Name != "" {
			if _, exists := config.GetJumpstarterInstances()[instance.Spec.JumpstarterInstanceRef.Name]; !exists {
				addError(sourceFile, fmt.Sprintf("ExporterInstance %s references non-existent jumpstarter instance %s",
					name, instance.Spec.JumpstarterInstanceRef.Name))
			}
		}

		// Check ConfigTemplateRef
		if instance.Spec.ConfigTemplateRef.Name != "" {
			if _, exists := config.GetExporterConfigTemplates()[instance.Spec.ConfigTemplateRef.Name]; !exists {
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
