// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
)

// ValidateTarget validates a Target
func ValidateTarget(target *lsv1alpha1.Target) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateTargetSpec(&target.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateTargetSpec(spec *lsv1alpha1.TargetSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.Configuration != nil && spec.SecretRef != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, spec, "either config or secretRef may be set, not both"))
	}

	return allErrs
}
