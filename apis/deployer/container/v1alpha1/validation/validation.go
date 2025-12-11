// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openmcp-project/landscaper/apis/core/validation"
	containerv1alpha1 "github.com/openmcp-project/landscaper/apis/deployer/container/v1alpha1"
	crval "github.com/openmcp-project/landscaper/apis/deployer/utils/continuousreconcile/validation"
)

// ValidateProviderConfiguration validates a container deployer configuration
func ValidateProviderConfiguration(config *containerv1alpha1.ProviderConfiguration) error {
	var allErrs field.ErrorList
	for i, secretRef := range config.RegistryPullSecrets {
		allErrs = append(allErrs, validation.ValidateObjectReference(secretRef, field.NewPath("registryPullSecrets").Index(i))...)
	}

	allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(field.NewPath("continuousReconcile"), config.ContinuousReconcile)...)
	return allErrs.ToAggregate()
}
