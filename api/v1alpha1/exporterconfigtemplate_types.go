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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExporterConfigTemplateSpec defines the desired state of ExporterConfigTemplate.
type ExporterConfigTemplateSpec struct {
	// ContainerImage specifies the container image to use for the exporter.
	// +kubebuilder:validation:Required
	ContainerImage string `json:"containerImage"`

	// ExporterMetadata defines metadata for the exporter itself.
	// +kubebuilder:validation:Required
	ExporterMetadata ExporterMeta `json:"exporterMetadata"`

	// ConfigTemplate is the raw YAML string content for the exporter's configuration file.
	// This content will be parsed by the component that uses this template.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ConfigTemplate string `json:"configTemplate"`

	// SystemdContainerTemplate is the raw YAML string content for the systemd container config template.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	SystemdContainerTemplate string `json:"systemdContainerTemplate"`

	// SystemdServiceTemplate is the raw YAML string content for the systemd service config template.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	SystemdServiceTemplate string `json:"systemdServiceTemplate"`
}

// ExporterMeta defines metadata for the exporter.
type ExporterMeta struct {
	// Name is the name of the exporter.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Labels are key-value pairs that are applied to the exporter.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ExporterConfigTemplateStatus defines the observed state of ExporterConfigTemplate.
type ExporterConfigTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ExporterConfigTemplate is the Schema for the exporterconfigtemplates API.
type ExporterConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExporterConfigTemplateSpec   `json:"spec,omitempty"`
	Status ExporterConfigTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ExporterConfigTemplateList contains a list of ExporterConfigTemplate.
type ExporterConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExporterConfigTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExporterConfigTemplate{}, &ExporterConfigTemplateList{})
}
