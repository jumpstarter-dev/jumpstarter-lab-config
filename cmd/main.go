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
	"os"

	"github.com/spf13/cobra"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	metav1alpha1 "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(metav1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

var rootCmd = &cobra.Command{
	Use:   "jumpstarter-lab-config",
	Short: "A tool for managing labs and environments with jumpstarter controllers",
	Long: `A gitops/configuration system and tool for managing labs and environments
	with jumpstarter controllers, exporter hosts (sidekicks), exporters, and clients
	in enterprise environments.`,
}

func init() {
	// Add subcommands to root
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(applyCmd)
}

// nolint:gocyclo
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
