/*
Copyright 2025. The Jumpstarter Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/loader"
)

var lintCmd = &cobra.Command{
	Use:   "lint [config-file]",
	Short: "Validate configuration files",
	Long:  `Lint and validate configuration files to ensure they are valid and follow the expected format.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPassFile, _ := cmd.Flags().GetString("vault-password-file")
		// Determine config file path
		configFilePath := "jumpstarter-lab.yaml" // default
		if len(args) > 0 {
			configFilePath = args[0]
		}

		// Load the configuration file
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			return fmt.Errorf("error loading config file %s: %w", configFilePath, err)
		}

		fmt.Println("Validating files from:")
		if len(cfg.Sources.Locations) > 0 {
			for _, pattern := range cfg.Sources.Locations {
				fmt.Printf("- %s\n", pattern)
			}
		}
		if len(cfg.Sources.Clients) > 0 {
			for _, pattern := range cfg.Sources.Clients {
				fmt.Printf("- %s\n", pattern)
			}
		}
		if len(cfg.Sources.ExporterHosts) > 0 {
			for _, pattern := range cfg.Sources.ExporterHosts {
				fmt.Printf("- %s\n", pattern)
			}
		}
		if len(cfg.Sources.Exporters) > 0 {
			for _, pattern := range cfg.Sources.Exporters {
				fmt.Printf("- %s\n", pattern)
			}
		}
		if len(cfg.Sources.ExporterTemplates) > 0 {
			for _, pattern := range cfg.Sources.ExporterTemplates {
				fmt.Printf("- %s\n", pattern)
			}
		}
		if len(cfg.Sources.JumpstarterInstances) > 0 {
			for _, pattern := range cfg.Sources.JumpstarterInstances {
				fmt.Printf("- %s\n", pattern)
			}
		}
		fmt.Println()

		// Initialize the loaded configuration structure
		loaded, err := loader.LoadAllResources(cfg, vaultPassFile)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		// Validate cross-references between objects
		errorsByFile := lint(loaded)
		if len(errorsByFile) > 0 {
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
			os.Exit(1)
		}

		keys := loaded.Variables.GetAllKeys()
		fmt.Printf("📚 Total Variables: %d\n", len(keys))
		fmt.Println("")
		fmt.Println("✅ All configurations are valid")
		return nil
	},
}

func init() {
	// Add the vault password file flag
	lintCmd.Flags().String("vault-password-file", "", "Path to the vault password file for decrypting variables")
	// Add the lint command to the root command
	rootCmd.AddCommand(lintCmd)
}

func lint(loaded *loader.LoadedLabConfig) map[string][]string {
	// This function is a placeholder for the linting logic.
	// Currently, it only validates cross-references between objects.
	// The actual linting logic can be implemented here as needed.
	return validateReferences(loaded)
}

// validateReferences checks that all cross-references between objects are valid
func validateReferences(loaded *loader.LoadedLabConfig) map[string][]string {
	errorsByFile := make(map[string][]string)

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
			errorsByFile[sourceFile] = make([]string, 0)
		}
		errorsByFile[sourceFile] = append(errorsByFile[sourceFile], errorMsg)
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
