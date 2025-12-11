// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openmcp-project/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/openmcp-project/landscaper/apis/deployer/manifest/v1alpha2"
)

// Convert_v1alpha1_ProviderConfiguration_To_v1alpha2_ProviderConfiguration is an manual conversion function.
func Convert_v1alpha1_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in *ProviderConfiguration, out *manifestv1alpha2.ProviderConfiguration, s conversion.Scope) error {
	out.UpdateStrategy = manifestv1alpha2.UpdateStrategy(in.UpdateStrategy)
	out.ReadinessChecks = in.ReadinessChecks
	if in.Manifests != nil {
		in, out := &in.Manifests, &out.Manifests
		*out = make([]managedresource.Manifest, len(*in))
		for i := range *in {
			(*out)[i] = managedresource.Manifest{
				Policy:   managedresource.ManagePolicy,
				Manifest: (*in)[i],
			}
		}
	} else {
		out.Manifests = nil
	}
	return nil
}

// Convert_v1alpha2_ProviderConfiguration_To_v1alpha1_ProviderConfiguration is an manual conversion function.
func Convert_v1alpha2_ProviderConfiguration_To_v1alpha1_ProviderConfiguration(in *manifestv1alpha2.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	out.UpdateStrategy = UpdateStrategy(in.UpdateStrategy)
	out.ReadinessChecks = in.ReadinessChecks
	if in.Manifests != nil {
		in, out := &in.Manifests, &out.Manifests
		*out = make([]*runtime.RawExtension, len(*in))
		for i := range *in {
			(*out)[i] = (*in)[i].Manifest
		}
	} else {
		out.Manifests = nil
	}
	return nil
}

// Convert_v1alpha1_ProviderStatus_To_v1alph2_ProviderStatus is an manual conversion function.
func Convert_v1alpha1_ProviderStatus_To_v1alph2_ProviderStatus(in *ProviderStatus, out *manifestv1alpha2.ProviderStatus, s conversion.Scope) error {
	if in.ManagedResources != nil {
		in, out := &in.ManagedResources, &out.ManagedResources
		*out = make([]managedresource.ManagedResourceStatus, len(*in))
		for i := range *in {
			tmp := (*in)[i]
			(*out)[i] = managedresource.ManagedResourceStatus{
				Policy: managedresource.ManagePolicy,
				Resource: corev1.ObjectReference{
					APIVersion: tmp.APIVersion,
					Kind:       tmp.Kind,
					Name:       tmp.Name,
					Namespace:  tmp.Namespace,
				},
			}
		}
	} else {
		out.ManagedResources = nil
	}
	return nil
}

// Convert_v1alpha2_ProviderStatus_To_v1alpha1_ProviderStatus is an manual conversion function.
func Convert_v1alpha2_ProviderStatus_To_v1alpha1_ProviderStatus(in *manifestv1alpha2.ProviderStatus, out *ProviderStatus, s conversion.Scope) error {
	if in.ManagedResources != nil {
		in, out := &in.ManagedResources, &out.ManagedResources
		*out = make([]lsv1alpha1.TypedObjectReference, len(*in))
		for i := range *in {
			res := (*in)[i].Resource
			(*out)[i] = lsv1alpha1.TypedObjectReference{
				APIVersion: res.APIVersion,
				Kind:       res.Kind,
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      res.Name,
					Namespace: res.Namespace,
				},
			}
		}
	} else {
		out.ManagedResources = nil
	}
	return nil
}
