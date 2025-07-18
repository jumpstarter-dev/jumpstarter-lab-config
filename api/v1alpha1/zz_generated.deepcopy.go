//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigTemplateRef) DeepCopyInto(out *ConfigTemplateRef) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigTemplateRef.
func (in *ConfigTemplateRef) DeepCopy() *ConfigTemplateRef {
	if in == nil {
		return nil
	}
	out := new(ConfigTemplateRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Contact) DeepCopyInto(out *Contact) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Contact.
func (in *Contact) DeepCopy() *Contact {
	if in == nil {
		return nil
	}
	out := new(Contact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DutLocationRef) DeepCopyInto(out *DutLocationRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DutLocationRef.
func (in *DutLocationRef) DeepCopy() *DutLocationRef {
	if in == nil {
		return nil
	}
	out := new(DutLocationRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterConfigTemplate) DeepCopyInto(out *ExporterConfigTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterConfigTemplate.
func (in *ExporterConfigTemplate) DeepCopy() *ExporterConfigTemplate {
	if in == nil {
		return nil
	}
	out := new(ExporterConfigTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterConfigTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterConfigTemplateList) DeepCopyInto(out *ExporterConfigTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExporterConfigTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterConfigTemplateList.
func (in *ExporterConfigTemplateList) DeepCopy() *ExporterConfigTemplateList {
	if in == nil {
		return nil
	}
	out := new(ExporterConfigTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterConfigTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterConfigTemplateSpec) DeepCopyInto(out *ExporterConfigTemplateSpec) {
	*out = *in
	in.ExporterMetadata.DeepCopyInto(&out.ExporterMetadata)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterConfigTemplateSpec.
func (in *ExporterConfigTemplateSpec) DeepCopy() *ExporterConfigTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(ExporterConfigTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterConfigTemplateStatus) DeepCopyInto(out *ExporterConfigTemplateStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterConfigTemplateStatus.
func (in *ExporterConfigTemplateStatus) DeepCopy() *ExporterConfigTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(ExporterConfigTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterHost) DeepCopyInto(out *ExporterHost) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterHost.
func (in *ExporterHost) DeepCopy() *ExporterHost {
	if in == nil {
		return nil
	}
	out := new(ExporterHost)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterHost) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterHostList) DeepCopyInto(out *ExporterHostList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExporterHost, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterHostList.
func (in *ExporterHostList) DeepCopy() *ExporterHostList {
	if in == nil {
		return nil
	}
	out := new(ExporterHostList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterHostList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterHostRef) DeepCopyInto(out *ExporterHostRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterHostRef.
func (in *ExporterHostRef) DeepCopy() *ExporterHostRef {
	if in == nil {
		return nil
	}
	out := new(ExporterHostRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterHostSpec) DeepCopyInto(out *ExporterHostSpec) {
	*out = *in
	out.LocationRef = in.LocationRef
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Power = in.Power
	out.Management = in.Management
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterHostSpec.
func (in *ExporterHostSpec) DeepCopy() *ExporterHostSpec {
	if in == nil {
		return nil
	}
	out := new(ExporterHostSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterHostStatus) DeepCopyInto(out *ExporterHostStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterHostStatus.
func (in *ExporterHostStatus) DeepCopy() *ExporterHostStatus {
	if in == nil {
		return nil
	}
	out := new(ExporterHostStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterInstance) DeepCopyInto(out *ExporterInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterInstance.
func (in *ExporterInstance) DeepCopy() *ExporterInstance {
	if in == nil {
		return nil
	}
	out := new(ExporterInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterInstance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterInstanceList) DeepCopyInto(out *ExporterInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExporterInstance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterInstanceList.
func (in *ExporterInstanceList) DeepCopy() *ExporterInstanceList {
	if in == nil {
		return nil
	}
	out := new(ExporterInstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExporterInstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterInstanceSpec) DeepCopyInto(out *ExporterInstanceSpec) {
	*out = *in
	out.DutLocationRef = in.DutLocationRef
	out.ExporterHostRef = in.ExporterHostRef
	out.JumpstarterInstanceRef = in.JumpstarterInstanceRef
	in.ConfigTemplateRef.DeepCopyInto(&out.ConfigTemplateRef)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterInstanceSpec.
func (in *ExporterInstanceSpec) DeepCopy() *ExporterInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(ExporterInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterInstanceStatus) DeepCopyInto(out *ExporterInstanceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterInstanceStatus.
func (in *ExporterInstanceStatus) DeepCopy() *ExporterInstanceStatus {
	if in == nil {
		return nil
	}
	out := new(ExporterInstanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterMeta) DeepCopyInto(out *ExporterMeta) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterMeta.
func (in *ExporterMeta) DeepCopy() *ExporterMeta {
	if in == nil {
		return nil
	}
	out := new(ExporterMeta)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JumpstarterInstance) DeepCopyInto(out *JumpstarterInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JumpstarterInstance.
func (in *JumpstarterInstance) DeepCopy() *JumpstarterInstance {
	if in == nil {
		return nil
	}
	out := new(JumpstarterInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JumpstarterInstance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JumpstarterInstanceList) DeepCopyInto(out *JumpstarterInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]JumpstarterInstance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JumpstarterInstanceList.
func (in *JumpstarterInstanceList) DeepCopy() *JumpstarterInstanceList {
	if in == nil {
		return nil
	}
	out := new(JumpstarterInstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JumpstarterInstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JumpstarterInstanceSpec) DeepCopyInto(out *JumpstarterInstanceSpec) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JumpstarterInstanceSpec.
func (in *JumpstarterInstanceSpec) DeepCopy() *JumpstarterInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(JumpstarterInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JumpstarterInstanceStatus) DeepCopyInto(out *JumpstarterInstanceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JumpstarterInstanceStatus.
func (in *JumpstarterInstanceStatus) DeepCopy() *JumpstarterInstanceStatus {
	if in == nil {
		return nil
	}
	out := new(JumpstarterInstanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JumsptarterInstanceRef) DeepCopyInto(out *JumsptarterInstanceRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JumsptarterInstanceRef.
func (in *JumsptarterInstanceRef) DeepCopy() *JumsptarterInstanceRef {
	if in == nil {
		return nil
	}
	out := new(JumsptarterInstanceRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LocationRef) DeepCopyInto(out *LocationRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocationRef.
func (in *LocationRef) DeepCopy() *LocationRef {
	if in == nil {
		return nil
	}
	out := new(LocationRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Management) DeepCopyInto(out *Management) {
	*out = *in
	out.SSH = in.SSH
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Management.
func (in *Management) DeepCopy() *Management {
	if in == nil {
		return nil
	}
	out := new(Management)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhysicalLocation) DeepCopyInto(out *PhysicalLocation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhysicalLocation.
func (in *PhysicalLocation) DeepCopy() *PhysicalLocation {
	if in == nil {
		return nil
	}
	out := new(PhysicalLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PhysicalLocation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhysicalLocationList) DeepCopyInto(out *PhysicalLocationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PhysicalLocation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhysicalLocationList.
func (in *PhysicalLocationList) DeepCopy() *PhysicalLocationList {
	if in == nil {
		return nil
	}
	out := new(PhysicalLocationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PhysicalLocationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhysicalLocationSpec) DeepCopyInto(out *PhysicalLocationSpec) {
	*out = *in
	if in.Contacts != nil {
		in, out := &in.Contacts, &out.Contacts
		*out = make([]Contact, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhysicalLocationSpec.
func (in *PhysicalLocationSpec) DeepCopy() *PhysicalLocationSpec {
	if in == nil {
		return nil
	}
	out := new(PhysicalLocationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhysicalLocationStatus) DeepCopyInto(out *PhysicalLocationStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhysicalLocationStatus.
func (in *PhysicalLocationStatus) DeepCopy() *PhysicalLocationStatus {
	if in == nil {
		return nil
	}
	out := new(PhysicalLocationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Power) DeepCopyInto(out *Power) {
	*out = *in
	out.SNMP = in.SNMP
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Power.
func (in *Power) DeepCopy() *Power {
	if in == nil {
		return nil
	}
	out := new(Power)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SNMPPower) DeepCopyInto(out *SNMPPower) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SNMPPower.
func (in *SNMPPower) DeepCopy() *SNMPPower {
	if in == nil {
		return nil
	}
	out := new(SNMPPower)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SSHCredentials) DeepCopyInto(out *SSHCredentials) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SSHCredentials.
func (in *SSHCredentials) DeepCopy() *SSHCredentials {
	if in == nil {
		return nil
	}
	out := new(SSHCredentials)
	in.DeepCopyInto(out)
	return out
}
