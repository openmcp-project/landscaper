// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openmcp-project/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	manifestv1alpha1 "github.com/openmcp-project/landscaper/apis/deployer/manifest/v1alpha1"
	manifestv1alpha2 "github.com/openmcp-project/landscaper/apis/deployer/manifest/v1alpha2"
)

var _ = Describe("Conversion", func() {

	Context("ProviderConfiguration", func() {

		var (
			v1alpha1Config = &manifestv1alpha1.ProviderConfiguration{
				UpdateStrategy: manifestv1alpha1.UpdateStrategyPatch,
				Manifests: []*runtime.RawExtension{
					{
						Raw: []byte("manifest1"),
					},
					{
						Raw: []byte("manifest2"),
					},
				},
			}

			manifestConfig = &manifestv1alpha2.ProviderConfiguration{
				UpdateStrategy: manifestv1alpha2.UpdateStrategyPatch,
				Manifests: []managedresource.Manifest{
					{
						Policy: managedresource.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest1"),
						},
					},
					{
						Policy: managedresource.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest2"),
						},
					},
				},
			}
		)

		Context("v1alpha1 to v1alpha2", func() {
			It("should convert all configuration and default the policy", func() {
				res := &manifestv1alpha2.ProviderConfiguration{}
				Expect(manifestv1alpha1.Convert_v1alpha1_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(v1alpha1Config, res, nil)).To(Succeed())
				Expect(res).To(Equal(manifestConfig))
			})
		})

		Context("v1alpha2 to v1alpha1", func() {
			It("should convert all configuration and default the policy", func() {
				res := &manifestv1alpha1.ProviderConfiguration{}
				Expect(manifestv1alpha1.Convert_v1alpha2_ProviderConfiguration_To_v1alpha1_ProviderConfiguration(manifestConfig, res, nil)).To(Succeed())
				Expect(res).To(Equal(v1alpha1Config))
			})
		})
	})

	Context("ProviderStatus", func() {

		var (
			v1alpha1Status = &manifestv1alpha1.ProviderStatus{
				ManagedResources: []lsv1alpha1.TypedObjectReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      "s1",
							Namespace: "default",
						},
					},
					{
						APIVersion: "v1",
						Kind:       "Secret",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      "s2",
							Namespace: "default",
						},
					},
				},
			}

			manifestStatus = &manifestv1alpha2.ProviderStatus{
				ManagedResources: []managedresource.ManagedResourceStatus{
					{
						Policy: managedresource.ManagePolicy,
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "s1",
							Namespace:  "default",
						},
					},
					{
						Policy: managedresource.ManagePolicy,
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "s2",
							Namespace:  "default",
						},
					},
				},
			}
		)

		It("v1alpha1 to v1alpha2", func() {
			res := &manifestv1alpha2.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_v1alpha1_ProviderStatus_To_v1alph2_ProviderStatus(v1alpha1Status, res, nil)).To(Succeed())
			Expect(res).To(Equal(manifestStatus))
		})

		It("v1alpha2 to v1alpha1", func() {
			res := &manifestv1alpha1.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_v1alpha2_ProviderStatus_To_v1alpha1_ProviderStatus(manifestStatus, res, nil)).To(Succeed())
			Expect(res).To(Equal(v1alpha1Status))
		})
	})

})
