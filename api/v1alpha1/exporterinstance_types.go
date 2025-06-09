/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExporterInstanceSpec defines the desired state of ExporterInstance.
type ExporterInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Type                   string                 `json:"type,omitempty"`
	Username               string                 `json:"username,omitempty"`
	DutLocationRef         DutLocationRef         `json:"dutLocationRef,omitempty"`
	ExporterHostRef        ExporterHostRef        `json:"exporterHostRef,omitempty"`
	JumpstarterInstanceRef JumsptarterInstanceRef `json:"jumpstarterInstanceRef,omitempty"`
	ConfigTemplateRef      ConfigTemplateRef      `json:"configTemplateRef,omitempty"`
	Labels                 map[string]string      `json:"labels,omitempty"`
	Notes                  string                 `json:"notes,omitempty"`
}

// DutLocationRef defines the location of the Device Under Test.
type DutLocationRef struct {
	Name string `json:"name,omitempty"`
	Rack string `json:"rack,omitempty"`
	Tray string `json:"tray,omitempty"`
}

// ExporterHostRef defines the reference to the exporter host.
type ExporterHostRef struct {
	Name string `json:"name,omitempty"`
}

// ControllerRef defines the reference to a controller.
type JumsptarterInstanceRef struct {
	Name string `json:"name,omitempty"`
}

// ConfigTemplateRef defines the reference to a configuration template.
type ConfigTemplateRef struct {
	Name       string            `json:"name,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ExporterInstanceStatus defines the observed state of ExporterInstance.
type ExporterInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ExporterInstance is the Schema for the exporterinstances API.
type ExporterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExporterInstanceSpec   `json:"spec,omitempty"`
	Status ExporterInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExporterInstanceList contains a list of ExporterInstance.
type ExporterInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExporterInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExporterInstance{}, &ExporterInstanceList{})
}
