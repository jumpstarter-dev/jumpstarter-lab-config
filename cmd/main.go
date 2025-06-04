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

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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

// nolint:gocyclo
func main() {
	// Load example
	obj, err := readAndDecodeYAML("example/devices/on-lab/ti-jacinto-j784s4xevm-01/ti-jacinto-j784s4xevm-01-sidekick.yaml")
	if err != nil {
		fmt.Printf("Error reading and decoding YAML: %v\n", err)
		os.Exit(1)
	}

	exporterHost, ok := obj.(*metav1alpha1.ExporterHost)
	if !ok {
		fmt.Printf("Decoded object is not an ExporterHost: %T\n", obj)
		os.Exit(1)
	}

	fmt.Printf("Successfully loaded ExporterHost: %+v\n", exporterHost)
	fmt.Printf("ExporterHost Name: %s\n", exporterHost.Name)
	fmt.Printf("ExporterHost ContainerImage: %s\n", exporterHost.Spec.ContainerImage)
	fmt.Printf("ExporterHost SNMP Host: %s\n", exporterHost.Spec.Power.SNMP.Host)

	obj, err = readAndDecodeYAML("example/devices/on-lab/ti-jacinto-j784s4xevm-01/ti-jacinto-j784s4xevm-01.yaml")
	if err != nil {
		fmt.Printf("Error reading and decoding YAML: %v\n", err)
		os.Exit(1)
	}

	exporterInstance := obj.(*metav1alpha1.ExporterInstance)
	if !ok {
		fmt.Printf("Decoded object is not an ExporterInstance: %T\n", obj)
		os.Exit(1)
	}
	fmt.Printf("Successfully loaded ExporterInstance: %+v\n", exporterInstance)
	fmt.Printf("ExporterInstance Name: %s\n", exporterInstance.Name)
	fmt.Printf("ExporterInstance Type: %s\n", exporterInstance.Spec.Type)
	fmt.Printf("ExporterInstance DutLocationRef: %+v\n", exporterInstance.Spec.DutLocationRef)
	fmt.Printf("ExporterInstance ExporterHostRef: %+v\n", exporterInstance.Spec.ExporterHostRef)

	obj, err = readAndDecodeYAML("example/exporter-templates/ti-am69/config.yaml")
	if err != nil {
		fmt.Printf("Error reading and decoding YAML: %v\n", err)
		os.Exit(1)
	}

	exporterConfigTemplate := obj.(*metav1alpha1.ExporterConfigTemplate)
	if !ok {
		fmt.Printf("Decoded object is not an ExporterConfigTemplate: %T\n", obj)
		os.Exit(1)
	}
	fmt.Printf("Successfully loaded ExporterInstance: %+v\n", exporterConfigTemplate)

}

func readAndDecodeYAML(filePath string) (runtime.Object, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}
	codecFactory := serializer.NewCodecFactory(scheme, serializer.EnableStrict)
	decode := codecFactory.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding YAML: %w", err)
	}

	return obj, nil
}
