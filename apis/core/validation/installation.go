// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"regexp"

	"github.com/robfig/cron/v3"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	"github.com/openmcp-project/landscaper/apis/core/v1alpha1/helper"
)

// InstallationNameMaxLength is the max allowed length of an installation name
const InstallationNameMaxLength = validation.DNS1123LabelMaxLength - len(helper.InstallationPrefix)

// InstallationGenerateNameMaxLength is the max length of an installation name minus the number of random characters kubernetes uses to generate a unique name
const InstallationGenerateNameMaxLength = InstallationNameMaxLength - 5

var targetMapKeyRegExp = regexp.MustCompile("^[a-z0-9]([a-z0-9.-]{0,61}[a-z0-9])?$")

// ValidateInstallation validates an Installation
func ValidateInstallation(inst *lsv1alpha1.Installation) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateInstallationObjectMeta(&inst.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateInstallationSpec(&inst.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateInstallationObjectMeta(objMeta *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(objMeta, true, apivalidation.NameIsDNSLabel, fldPath)...)

	if len(objMeta.GetName()) > InstallationNameMaxLength {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), objMeta.GetName(), validation.MaxLenError(InstallationNameMaxLength)))
	} else if len(objMeta.GetGenerateName()) > InstallationGenerateNameMaxLength {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("generateName"), objMeta.GetGenerateName(), validation.MaxLenError(InstallationGenerateNameMaxLength)))
	}

	return allErrs
}

// ValidateInstallationSpec validates the spec of an Installation
func ValidateInstallationSpec(spec *lsv1alpha1.InstallationSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationImports(spec.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(spec.Exports, fldPath.Child("exports"))...)

	// check Blueprint and ComponentDescriptor
	allErrs = append(allErrs, ValidateInstallationBlueprint(spec.Blueprint, fldPath.Child("blueprint"))...)
	allErrs = append(allErrs, ValidateInstallationComponentDescriptor(spec.ComponentDescriptor, fldPath.Child("componentDescriptor"))...)

	allErrs = append(allErrs, ValidateInstallationAutomaticReconcile(spec.AutomaticReconcile, fldPath.Child("automaticReconcile"))...)

	return allErrs
}

// ValidateInstallationBlueprint validates the Blueprint definition of an Installation
func ValidateInstallationBlueprint(bp lsv1alpha1.BlueprintDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// check that either inline blueprint or reference is provided (and not both)
	allErrs = append(allErrs, ValidateExactlyOneOf(fldPath.Child("definition"), bp, "Inline", "Reference")...)

	return allErrs
}

// ValidateInstallationComponentDescriptor validates the ComponentDesriptor of an Installation
func ValidateInstallationComponentDescriptor(cd *lsv1alpha1.ComponentDescriptorDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// check that a ComponentDescriptor - if given - is either inline or ref but not both
	if cd != nil {
		allErrs = append(allErrs, ValidateExactlyOneOf(fldPath.Child("definition"), *cd, "Inline", "Reference")...)
	}

	return allErrs
}

func ValidateInstallationAutomaticReconcile(automaticReconcile *lsv1alpha1.AutomaticReconcile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if automaticReconcile != nil {
		allErrs = append(allErrs, ValidateInstallationSucceededReconcile(automaticReconcile.SucceededReconcile, fldPath.Child("succeededReconcile"))...)
		allErrs = append(allErrs, ValidateInstallationFailedReconcile(automaticReconcile.FailedReconcile, fldPath.Child("failedReconcile"))...)
	}

	return allErrs
}

func ValidateInstallationFailedReconcile(failedReconcile *lsv1alpha1.FailedReconcile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if failedReconcile != nil {
		if failedReconcile.CronSpec != "" {
			_, err := cron.ParseStandard(failedReconcile.CronSpec)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("cronSpec"), failedReconcile.CronSpec,
					"field must be a valid cron spec"))
			}
		}
	}

	return allErrs
}

func ValidateInstallationSucceededReconcile(succeededReconcile *lsv1alpha1.SucceededReconcile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if succeededReconcile != nil {
		if succeededReconcile.CronSpec != "" {
			_, err := cron.ParseStandard(succeededReconcile.CronSpec)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("cronSpec"), succeededReconcile.CronSpec,
					"field must be a valid cron spec"))
			}
		}
	}

	return allErrs
}

// ValidateInstallationImports validates the imports of an Installation
func ValidateInstallationImports(imports lsv1alpha1.InstallationImports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	importNames := sets.NewString()
	var tmpErrs field.ErrorList

	tmpErrs, importNames = ValidateInstallationDataImports(imports.Data, fldPath.Child("data"), importNames)
	allErrs = append(allErrs, tmpErrs...)
	tmpErrs, _ = ValidateInstallationTargetImports(imports.Targets, fldPath.Child("targets"), importNames)
	allErrs = append(allErrs, tmpErrs...)

	return allErrs
}

// ValidateInstallationDataImports validates the data imports of an Installation
func ValidateInstallationDataImports(imports []lsv1alpha1.DataImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	allErrs := field.ErrorList{}

	for idx, imp := range imports {
		impPath := fldPath.Index(idx)

		allErrs = append(allErrs, ValidateExactlyOneOf(impPath, imp, "DataRef", "SecretRef", "ConfigMapRef")...)

		if imp.SecretRef != nil {
			allErrs = append(allErrs, ValidateLocalSecretReference(*imp.SecretRef, impPath.Child("secretRef"))...)
		}

		if imp.ConfigMapRef != nil {
			allErrs = append(allErrs, ValidateLocalConfigMapReference(*imp.ConfigMapRef, impPath.Child("configMapRef"))...)
		}

		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(impPath.Child("name"), "name must not be empty"))
			continue
		}
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(impPath, imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
}

// ValidateInstallationTargetImports validates the target imports of an Installation
func ValidateInstallationTargetImports(imports []lsv1alpha1.TargetImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	allErrs := field.ErrorList{}

	for idx, imp := range imports {
		fldPathIdx := fldPath.Index(idx)
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPathIdx.Child("name"), "name must not be empty"))
		}
		allErrs = append(allErrs, ValidateExactlyOneOf(fldPathIdx, imp, "Target", "Targets", "TargetMap", "TargetMapReference", "TargetListReference")...)
		if len(imp.Targets) > 0 {
			for idx2, tg := range imp.Targets {
				if len(tg) == 0 {
					allErrs = append(allErrs, field.Required(fldPathIdx.Child("targets").Index(idx2), "target must not be empty"))
				}
			}
		}
		if imp.TargetMap != nil {
			for key, tg := range imp.TargetMap {
				if !targetMapKeyRegExp.MatchString(key) {
					allErrs = append(allErrs, field.Invalid(fldPathIdx.Child("targetMap").Key(key), key,
						"key must contain only lower-case alphanumeric characters, dots, or dashes; "+
							"it must begin and end with a lower-case alphanumeric character; "+
							"it must not be empty, and not longer than 63 characters"))
				}
				if len(tg) == 0 {
					allErrs = append(allErrs, field.Required(fldPathIdx.Child("targetMap").Key(key), "target must not be empty"))
				}
			}
		}
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(fldPathIdx, imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
}

// ValidateInstallationExports validates the exports of an Installation
func ValidateInstallationExports(exports lsv1alpha1.InstallationExports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationDataExports(exports.Data, fldPath.Child("data"))...)
	allErrs = append(allErrs, ValidateInstallationTargetExports(exports.Targets, fldPath.Child("targets"))...)

	return allErrs
}

// ValidateInstallationDataExports validates the data exports of an Installation
func ValidateInstallationDataExports(exports []lsv1alpha1.DataExport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range exports {
		if imp.DataRef == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("dataRef"), "dataRef must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateInstallationTargetExports validates the target exports of an Installation
func ValidateInstallationTargetExports(exports []lsv1alpha1.TargetExport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range exports {
		if imp.Target == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("target"), "target must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateObjectReference validates that the object reference is valid
func ValidateObjectReference(or lsv1alpha1.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if or.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	if or.Namespace == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("namespace"), "namespace must not be empty"))
	}

	return allErrs
}

// ValidateObjectReferenceList validates a list of object references
func ValidateObjectReferenceList(orl []lsv1alpha1.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, e := range orl {
		allErrs = append(allErrs, ValidateObjectReference(e, fldPath.Index(i))...)
	}

	return allErrs
}

// ValidateLocalSecretReference validates that the local secret reference is valid
func ValidateLocalSecretReference(sr lsv1alpha1.LocalSecretReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if sr.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	return allErrs
}

// ValidateLocalConfigMapReference validates that the local configmap reference is valid
func ValidateLocalConfigMapReference(cmr lsv1alpha1.LocalConfigMapReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if cmr.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	return allErrs
}
