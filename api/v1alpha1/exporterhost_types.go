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

// ExporterHostSpec defines the desired state of ExporterHost.
type ExporterHostSpec struct {
	// LocationRef references the physical location of the exporter host.
	LocationRef LocationRef `json:"locationRef,omitempty"`

	// ContainerImage is the container image to be used for the exporter.
	ContainerImage string `json:"containerImage,omitempty"`

	// Addresses is a list of network addresses for the exporter host.
	Addresses []string `json:"addresses,omitempty"`

	// Power defines the power control configuration for the exporter host.
	Power Power `json:"power,omitempty"`

	// Management options for the exporter host, could be SSH access, flightctl device ids, etc..
	Management Management `json:"management,omitempty"`
}

type Management struct {
	SSH SSHCredentials `json:"ssh,omitempty"`
}

type SSHCredentials struct {
	// Host is the hostname or IP address for SSH access.
	Host string `json:"host,omitempty"`
	// User is the SSH username.
	User string `json:"user,omitempty"`
	// KeyFile is the path to the SSH private key file.
	KeyFile string `json:"keyFile,omitempty"`
	// SSHKeyData is the SSH private key data as a string.
	SSHKeyData string `json:"sshKeyData,omitempty"`
	// SSHKeyPassword is the password for encrypted SSH private keys.
	SSHKeyPassword string `json:"sshKeyPassword,omitempty"`
	// Password is the SSH password (if not using key-based auth).
	Password string `json:"password,omitempty"`
	// Port is the SSH port (default is 22).
	Port int `json:"port,omitempty"`
}

// LocationRef defines the physical location details.
type LocationRef struct {
	// Name is the name of the location (e.g., lab name).
	Name string `json:"name,omitempty"`
	// Rack is the rack identifier within the location.
	Rack string `json:"rack,omitempty"`
	// Tray is the tray identifier within the rack.
	Tray string `json:"tray,omitempty"`
}

// Power defines the power control configuration.
type Power struct {
	// SNMP defines the SNMP configuration for power control.
	SNMP SNMPPower `json:"snmp,omitempty"`
}

// SNMPPower defines SNMP specific power control parameters.
type SNMPPower struct {
	// Host is the hostname or IP address of the SNMP-enabled PDU.
	Host string `json:"host,omitempty"`
	// User is the SNMP username.
	User string `json:"user,omitempty"`
	// Password is the SNMP password.
	Password string `json:"password,omitempty"`
	// OID is the SNMP OID for controlling the power outlet.
	OID string `json:"oid,omitempty"`
	// Plug is the outlet/plug number on the PDU.
	Plug int `json:"plug,omitempty"`
}

// ExporterHostStatus defines the observed state of ExporterHost.
type ExporterHostStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ExporterHost is the Schema for the exporterhosts API.
type ExporterHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExporterHostSpec   `json:"spec,omitempty"`
	Status ExporterHostStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ExporterHostList contains a list of ExporterHost.
type ExporterHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExporterHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExporterHost{}, &ExporterHostList{})
}
