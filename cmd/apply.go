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

	"github.com/spf13/cobra"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/loader"
)

var applyCmd = &cobra.Command{
	Use:   "apply [config-file]",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes to the jumpstarter controllers. Use --dry-run to verify changes before applying.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		prune, _ := cmd.Flags().GetBool("prune")

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

		fmt.Println("Reading files from:")
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
		_, err = loader.LoadAllResources(cfg)
		if err != nil {
			return fmt.Errorf("error loading resources: %w", err)
		}

		fmt.Println("✅ All configurations are valid")
		fmt.Println()

		if dryRun {
			fmt.Println("Detected changes to apply:")
			fmt.Println()
			// TODO: Implement dry-run logic to detect changes
			fmt.Println("⚠️ Dry-run mode: no changes will be applied")
		} else {
			fmt.Println("Applying changes:")
			fmt.Println()
			// TODO: Implement actual apply logic
			if prune {
				fmt.Println("⚠️ Prune mode enabled: unused resources will be deleted")
			}
		}

		return nil
	},
}

func init() {
	// Add flags to apply command
	applyCmd.Flags().Bool("dry-run", false, "Show what would be applied without making changes")
	applyCmd.Flags().Bool("prune", false, "Delete resources that are no longer defined in configuration")
}
