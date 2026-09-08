// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/oci/artdesc"
	ocicpi "ocm.software/ocm/api/oci/cpi"
	"ocm.software/ocm/api/oci/extensions/repositories/ocireg"
	"ocm.software/ocm/api/ocm/cpi/repocpi"
	"ocm.software/ocm/api/ocm/extensions/repositories/genericocireg"

	"github.com/openmcp-project/landscaper/pkg/components/model"
	"github.com/openmcp-project/landscaper/pkg/components/ocmlib"
)

// ImageReference is an OCI reference split into the parts a template usually needs.
// At least one of Tag and Digest is set.
type ImageReference struct {
	// Reference is the full OCI reference without scheme, e.g. "ghcr.io/org/image:1.0.0".
	// A reference that is tagged and pinned by digest keeps both.
	Reference string `json:"reference"`
	// Repository is the reference without tag or digest, e.g. "ghcr.io/org/image".
	Repository string `json:"repository"`
	// Tag is set when the reference is tagged.
	Tag string `json:"tag"`
	// Digest is set when the reference is pinned by digest.
	Digest string `json:"digest"`
}

// ResolveImageReference turns the access of a resource into an OCI reference.
//
// A localBlob is addressed inside the OCI repository that holds the component version itself:
//
//	<OCI repository of cv>@<localReference>
//
// Every other access type must carry an imageReference, which is used as it is.
func ResolveImageReference(cv model.ComponentVersion, access map[string]interface{}) (ImageReference, error) {
	if access == nil {
		return ImageReference{}, errors.New("resource has no access")
	}
	accessType, _ := access["type"].(string)

	var ref string
	var err error
	if isLocalBlob(accessType) {
		ref, err = localBlobImageReference(cv, access)
	} else {
		ref, err = ociImageReference(access)
	}
	if err != nil {
		return ImageReference{}, err
	}

	return parseImageReference(ref)
}

// The access type of a local blob and its spellings, as registered in the descriptor
// scheme of open-component-model/bindings/go/descriptor/v2.
const (
	localBlobAccessType        = "LocalBlob"
	legacyLocalBlobAccessType  = "localBlob"
	localBlobAccessTypeVersion = "v1"
)

// isLocalBlob reports whether accessType is one of the spellings the OCM descriptor
// scheme accepts for a local blob: "LocalBlob/v1", "LocalBlob", "localBlob/v1", "localBlob".
// The comparison is exact, like the scheme's.
func isLocalBlob(accessType string) bool {
	switch accessType {
	case localBlobAccessType + "/" + localBlobAccessTypeVersion, localBlobAccessType,
		legacyLocalBlobAccessType + "/" + localBlobAccessTypeVersion, legacyLocalBlobAccessType:
		return true
	}
	return false
}

func ociImageReference(access map[string]interface{}) (string, error) {
	ref, _ := access["imageReference"].(string)
	if ref == "" {
		return "", errors.New("access has no imageReference")
	}
	return ref, nil
}

// ErrNotInOCIRegistry is returned for a localBlob of a component version that does not
// live in an OCI registry, e.g. one read from a CTF or an inline descriptor.
var ErrNotInOCIRegistry = errors.New("component version is not stored in an OCI registry")

// localBlobImageReference addresses the blob inside the OCI repository that holds the
// component version: <base URL>/<namespace of the component>@<localReference>. Base URL
// and namespace come from the OCM library repository the version was loaded from.
//
// Only a blob stored as an OCI image manifest or index can be pulled by that reference.
// Anything else, such as the artifact set archives OCM v1 writes, is a plain layer.
func localBlobImageReference(cv model.ComponentVersion, access map[string]interface{}) (string, error) {
	mediaType, _ := access["mediaType"].(string)
	if mediaType != artdesc.MediaTypeImageManifest && mediaType != artdesc.MediaTypeImageIndex {
		return "", fmt.Errorf("localBlob with media type %q is not an OCI image manifest or index and cannot be pulled by digest", mediaType)
	}
	localReference, _ := access["localReference"].(string)
	if localReference == "" {
		return "", errors.New("localBlob access has no localReference")
	}
	ocmCV, ok := cv.(*ocmlib.ComponentVersion)
	if !ok {
		return "", ErrNotInOCIRegistry
	}
	// A failed lookup leaves impl nil, so the type assertion below reports it.
	impl, _ := repocpi.GetRepositoryImplementation(ocmCV.GetOCMObject().Repository())
	ocmRepo, ok := impl.(*genericocireg.RepositoryImpl)
	if !ok {
		return "", ErrNotInOCIRegistry
	}
	ociImpl, _ := ocicpi.GetRepositoryImplementation(ocmRepo.GetOCIRepository())
	ociRepo, ok := ociImpl.(*ocireg.RepositoryImpl) // a CTF is an OCI repository too, but not a registry
	if !ok {
		return "", ErrNotInOCIRegistry
	}
	namespace, err := ocmRepo.MapComponentNameToNamespace(cv.GetName())
	if err != nil {
		return "", err
	}
	// not path.Join: it would collapse the "//" of a scheme in the base URL
	return strings.TrimSuffix(ociRepo.GetBaseURL(), "/") + "/" + namespace + "@" + localReference, nil
}

func parseImageReference(ref string) (ImageReference, error) {
	spec, err := oci.ParseRef(ref)
	if err != nil {
		return ImageReference{}, fmt.Errorf("invalid image reference %q: %w", ref, err)
	}
	if spec.Tag == nil && spec.Digest == nil {
		return ImageReference{}, fmt.Errorf("image reference %q has neither tag nor digest", ref)
	}
	// Host and Repository carry no scheme, unlike spec.Name().
	parsed := ImageReference{Repository: path.Join(spec.Host, spec.Repository)}
	parsed.Reference = parsed.Repository
	if spec.Tag != nil {
		parsed.Tag = *spec.Tag
		parsed.Reference += ":" + parsed.Tag
	}
	if spec.Digest != nil {
		parsed.Digest = spec.Digest.String()
		parsed.Reference += "@" + parsed.Digest
	}
	return parsed, nil
}
