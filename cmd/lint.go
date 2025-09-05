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
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config_lint"
)

var lintCmd = &cobra.Command{
	Use:          "lint [config-file]",
	Short:        "Validate configuration files",
	Long:         `Lint and validate configuration files to ensure they are valid and follow the expected format.`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPassFile, _ := cmd.Flags().GetString("vault-password-file")
		cpuProfile, _ := cmd.Flags().GetString("cpu-profile")
		// Determine config file path
		configFilePath := defaultConfigFile
		if len(args) > 0 {
			configFilePath = args[0]
		}

		// Load the configuration file
		cfg, err := config.LoadConfig(configFilePath, vaultPassFile)
		if err != nil {
			return fmt.Errorf("error loading config file %s: %w", configFilePath, err)
		}

		fmt.Println("🔍 Validating configuration...")

		// Start CPU profiling if requested
		if cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				return fmt.Errorf("could not create CPU profile: %w", err)
			}
			defer func() {
				if closeErr := f.Close(); closeErr != nil {
					fmt.Printf("Warning: failed to close profile file: %v\n", closeErr)
				}
			}()

			if err := pprof.StartCPUProfile(f); err != nil {
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			defer pprof.StopCPUProfile()

			fmt.Printf("📊 CPU profiling enabled, output will be saved to: %s\n", cpuProfile)
		}

		// Time the validation
		start := time.Now()
		err = config_lint.ValidateWithError(cfg)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Configuration validation failed in %v\n", duration)
			return err
		}

		fmt.Printf("✅ Configuration validation completed in %v\n", duration)

		return nil
	},
}

func init() {
	// Add the vault password file flag
	lintCmd.Flags().String("vault-password-file", "", "Path to the vault password file for decrypting variables")
	// Add the CPU profiling flag
	lintCmd.Flags().String("cpu-profile", "", "Enable CPU profiling and save output to the specified file")
	// Add the lint command to the root command
	rootCmd.AddCommand(lintCmd)
}
