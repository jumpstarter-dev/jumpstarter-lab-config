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

// JumpstarterInstanceSpec defines the desired state of JumpstarterInstance.
type JumpstarterInstanceSpec struct {
	// KubeContext specifies the kubeconfig context to use for communicating with
	// the cluster where the Jumpstarter controller is running or targeting.
	// +kubebuilder:validation:Optional
	KubeContext string `json:"kube-context,omitempty"`

	// Kubeconfig specifies the kubeconfig to use for communicating with
	// the cluster where the Jumpstarter controller is running or targeting.
	// +kubebuilder:validation:Optional
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// Endpoints lists the gRPC endpoints for the Jumpstarter instance.
	// These are the addresses that clients will use to connect to the Jumpstarter services.
	// +kubebuilder:validation:Optional
	Endpoints []string `json:"endpoints,omitempty"` // MinItems=0 is implicit for optional array

	// Namespace specifies the Kubernetes namespace relevant to this JumpstarterInstance.
	// This could be the namespace where Jumpstarter components are deployed
	// or the namespace it primarily operates within.
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// Notes provides additional information or comments about the Jumpstarter instance.
	// This field can be used to document the purpose, configuration, or any other relevant details.
	// +kubebuilder:validation:Optional
	Notes string `json:"notes,omitempty"`
}

// JumpstarterInstanceStatus defines the observed state of JumpstarterInstance.
type JumpstarterInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JumpstarterInstance is the Schema for the jumpstarterinstances API.
type JumpstarterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JumpstarterInstanceSpec   `json:"spec,omitempty"`
	Status JumpstarterInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JumpstarterInstanceList contains a list of JumpstarterInstance.
type JumpstarterInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JumpstarterInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JumpstarterInstance{}, &JumpstarterInstanceList{})
}
