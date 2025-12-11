// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	"github.com/openmcp-project/landscaper/apis/core/validation"
)

var _ = Describe("Target", func() {
	Context("Spec", func() {

		It("should accept a Target with an empty spec", func() {
			t := &lsv1alpha1.Target{
				Spec: lsv1alpha1.TargetSpec{},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

		It("should reject a Target with secretRef and config set", func() {
			t := &lsv1alpha1.Target{
				Spec: lsv1alpha1.TargetSpec{
					Configuration: lsv1alpha1.NewAnyJSONPointer([]byte("foo")),
					SecretRef: &lsv1alpha1.LocalSecretReference{
						Name: "foo",
					},
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should accept a Target with a secretRef", func() {
			t := &lsv1alpha1.Target{
				Spec: lsv1alpha1.TargetSpec{
					SecretRef: &lsv1alpha1.LocalSecretReference{
						Name: "foo",
					},
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

		It("should accept a Target with an inline config", func() {
			t := &lsv1alpha1.Target{
				Spec: lsv1alpha1.TargetSpec{
					Configuration: lsv1alpha1.NewAnyJSONPointer([]byte("foo")),
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

	})
})
