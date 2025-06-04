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

// Contact defines contact information.
type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// PhysicalLocationSpec defines the desired state of PhysicalLocation.
type PhysicalLocationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Description string    `json:"description,omitempty"`
	Address     string    `json:"address,omitempty"`
	Contacts    []Contact `json:"contacts,omitempty"`
}

// PhysicalLocationStatus defines the observed state of PhysicalLocation.
type PhysicalLocationStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PhysicalLocation is the Schema for the physicallocations API.
type PhysicalLocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PhysicalLocationSpec   `json:"spec,omitempty"`
	Status PhysicalLocationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PhysicalLocationList contains a list of PhysicalLocation.
type PhysicalLocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PhysicalLocation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PhysicalLocation{}, &PhysicalLocationList{})
}
