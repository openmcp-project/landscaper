// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/oci/artdesc"
	ocicpi "ocm.software/ocm/api/oci/cpi"
	"ocm.software/ocm/api/oci/extensions/repositories/ocireg"
	"ocm.software/ocm/api/ocm/cpi/repocpi"
	ocihdlr "ocm.software/ocm/api/ocm/extensions/blobhandler/handlers/oci"
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

// ResolveImageReference turns the access of resource, a resource of cv as returned by
// getResource, into an OCI reference.
//
// A localBlob is addressed inside the OCI repository that holds the component version
// itself. Its localReference is the digest; the resource version serves as the tag, the
// same tag OCM gives the blob when it hands it out as an OCI layout:
//
//	<OCI repository of cv>:<resource version>@<localReference>
//
// Every other access type must carry an imageReference, which is used as it is.
func ResolveImageReference(cv model.ComponentVersion, resource map[string]interface{}) (ImageReference, error) {
	access, _ := resource["access"].(map[string]interface{})
	if access == nil {
		return ImageReference{}, errors.New("resource has no access")
	}
	accessType, _ := access["type"].(string)

	var ref string
	var err error
	if isLocalBlob(accessType) {
		version, _ := resource["version"].(string)
		ref, err = localBlobImageReference(cv, version, access)
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
// component version: <registry>/<namespace of the version>:<version>@<localReference>.
// The version is used as the tag as it is; the tag is left out when version is empty.
//
// Only a blob stored as an OCI image manifest or index can be pulled by that reference.
// Anything else, such as the artifact set archives OCM v1 writes, is a plain layer.
func localBlobImageReference(cv model.ComponentVersion, version string, access map[string]interface{}) (string, error) {
	mediaType, _ := access["mediaType"].(string)
	if mediaType != artdesc.MediaTypeImageManifest && mediaType != artdesc.MediaTypeImageIndex {
		return "", fmt.Errorf("localBlob with media type %q is not an OCI image manifest or index and cannot be pulled by digest", mediaType)
	}
	localReference, _ := access["localReference"].(string)
	blobDigest, err := digest.Parse(localReference)
	if err != nil {
		return "", fmt.Errorf("localReference %q is not a digest: %w", localReference, err)
	}
	baseURL, namespace, err := ociLocation(cv)
	if err != nil {
		return "", err
	}
	ref := fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), namespace)
	if version != "" {
		ref = fmt.Sprintf("%s:%s", ref, version)
	}
	return fmt.Sprintf("%s@%s", ref, blobDigest), nil
}

// ociLocation returns the base URL of the registry and the namespace the component
// version was read from. The base URL may carry a scheme, e.g. "http://localhost:5000".
func ociLocation(cv model.ComponentVersion) (string, string, error) {
	ocmCV, ok := cv.(*ocmlib.ComponentVersion)
	if !ok {
		return "", "", ErrNotInOCIRegistry
	}
	container, err := repocpi.GetComponentVersionImpl[*genericocireg.ComponentVersionContainer](ocmCV.GetOCMObject())
	if err != nil {
		return "", "", ErrNotInOCIRegistry
	}
	storage, ok := container.GetStorageContext().(*ocihdlr.StorageContext)
	if !ok {
		return "", "", ErrNotInOCIRegistry
	}
	// A failed lookup leaves impl nil, so the type assertion below reports it.
	impl, _ := ocicpi.GetRepositoryImplementation(storage.Repository)
	registry, ok := impl.(*ocireg.RepositoryImpl) // a CTF is an OCI repository too, but not a registry
	if !ok {
		return "", "", ErrNotInOCIRegistry
	}
	return registry.GetBaseURL(), storage.Namespace.GetNamespace(), nil
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
