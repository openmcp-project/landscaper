// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"context"
	"net/http/httptest"
	"strings"

	"github.com/google/go-containerregistry/pkg/registry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/ocm/api/datacontext"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/composition"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"

	"github.com/openmcp-project/landscaper/pkg/components/model"
	"github.com/openmcp-project/landscaper/pkg/components/ocmlib"
	"github.com/openmcp-project/landscaper/pkg/landscaper/installations/executions/template"
)

const (
	testDigest           = "sha256:66371f17cc61bbbed2667b0285a10981deba5eb969df9bfd4cf273706044ddcb"
	testComponentName    = "example.com/mycomp"
	testComponentVersion = "1.0.0"
	testResourceVersion  = "6.14.1"
)

var _ = Describe("ResolveImageReference", func() {

	// resource wraps an access into the shape getResource returns.
	resource := func(version string, access map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"name": "image", "version": version, "access": access}
	}

	localBlob := func(accessType string, extra map[string]interface{}) map[string]interface{} {
		access := map[string]interface{}{
			"type":           accessType,
			"localReference": testDigest,
			"mediaType":      "application/vnd.oci.image.manifest.v1+json",
		}
		for k, v := range extra {
			access[k] = v
		}
		return resource(testResourceVersion, access)
	}

	ociArtifact := func(accessType, ref string) map[string]interface{} {
		return resource(testResourceVersion, map[string]interface{}{"type": accessType, "imageReference": ref})
	}

	// ociCV returns a component version stored in an in-memory OCI registry and the
	// OCI repository it lives in.
	ociCV := func() (model.ComponentVersion, string) {
		cv, host := newOCIComponentVersion(nil)
		return cv, host + "/internal/component-descriptors/" + testComponentName
	}

	Context("ociArtifact access", func() {
		It("splits a tagged reference into repository and tag", func() {
			ref, err := template.ResolveImageReference(nil, ociArtifact("ociArtifact", "ghcr.io/example/myimage:1.0.0"))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal(template.ImageReference{
				Reference:  "ghcr.io/example/myimage:1.0.0",
				Repository: "ghcr.io/example/myimage",
				Tag:        "1.0.0",
			}))
		})

		It("splits a digest reference into repository and digest", func() {
			ref, err := template.ResolveImageReference(nil, ociArtifact("ociArtifact", "ghcr.io/example/myimage@"+testDigest))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal(template.ImageReference{
				Reference:  "ghcr.io/example/myimage@" + testDigest,
				Repository: "ghcr.io/example/myimage",
				Digest:     testDigest,
			}))
		})

		It("keeps both tag and digest of a pinned tagged reference", func() {
			ref, err := template.ResolveImageReference(nil, ociArtifact("ociArtifact", "ghcr.io/example/myimage:1.0.0@"+testDigest))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal(template.ImageReference{
				Reference:  "ghcr.io/example/myimage:1.0.0@" + testDigest,
				Repository: "ghcr.io/example/myimage",
				Tag:        "1.0.0",
				Digest:     testDigest,
			}))
		})

		It("accepts the legacy ociRegistry access type", func() {
			ref, err := template.ResolveImageReference(nil, ociArtifact("ociRegistry", "ghcr.io/example/myimage:1.0.0"))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Repository).To(Equal("ghcr.io/example/myimage"))
			Expect(ref.Tag).To(Equal("1.0.0"))
		})

		It("does not take the resource version as tag", func() {
			ref, err := template.ResolveImageReference(nil, ociArtifact("ociArtifact", "ghcr.io/example/myimage@"+testDigest))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Tag).To(BeEmpty())
		})

		It("fails without an imageReference", func() {
			_, err := template.ResolveImageReference(nil, resource(testResourceVersion, map[string]interface{}{"type": "ociArtifact"}))
			Expect(err).To(MatchError(ContainSubstring("imageReference")))
		})
	})

	Context("localBlob access", func() {
		It("builds the reference from the OCI repository of the component version, without scheme", func() {
			cv, repo := ociCV()
			ref, err := template.ResolveImageReference(cv, localBlob("LocalBlob/v1", nil))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal(template.ImageReference{
				Reference:  repo + ":" + testResourceVersion + "@" + testDigest,
				Repository: repo,
				Tag:        testResourceVersion,
				Digest:     testDigest,
			}))
		})

		It("leaves the tag empty without a resource version", func() {
			cv, repo := ociCV()
			ref, err := template.ResolveImageReference(cv, resource("", localBlob("LocalBlob/v1", nil)["access"].(map[string]interface{})))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal(template.ImageReference{
				Reference:  repo + "@" + testDigest,
				Repository: repo,
				Digest:     testDigest,
			}))
		})

		It("fails when the resource version is not a valid tag", func() {
			cv, _ := ociCV()
			_, err := template.ResolveImageReference(cv, resource("1.0.0+build.7", localBlob("LocalBlob/v1", nil)["access"].(map[string]interface{})))
			Expect(err).To(MatchError(ContainSubstring("invalid image reference")))
		})

		It("accepts every spelling the OCM scheme registers for a local blob", func() {
			cv, _ := ociCV()
			for _, accessType := range []string{"LocalBlob/v1", "LocalBlob", "localBlob/v1", "localBlob"} {
				ref, err := template.ResolveImageReference(cv, localBlob(accessType, nil))
				Expect(err).ToNot(HaveOccurred(), accessType)
				Expect(ref.Digest).To(Equal(testDigest), accessType)
			}
		})

		It("does not treat other spellings as a local blob", func() {
			for _, accessType := range []string{"localblob", "LocalBlob/v2", "LOCALBLOB"} {
				_, err := template.ResolveImageReference(nil, localBlob(accessType, nil))
				Expect(err).To(MatchError(ContainSubstring("imageReference")), accessType)
			}
		})

		It("fails when the component version is not stored in an OCI registry", func() {
			_, err := template.ResolveImageReference(newNonOCIComponentVersion(), localBlob("LocalBlob/v1", nil))
			Expect(err).To(MatchError(template.ErrNotInOCIRegistry))
		})

		It("fails without a component version", func() {
			_, err := template.ResolveImageReference(nil, localBlob("LocalBlob/v1", nil))
			Expect(err).To(MatchError(template.ErrNotInOCIRegistry))
		})

		It("accepts an image index", func() {
			cv, repo := ociCV()
			ref, err := template.ResolveImageReference(cv, localBlob("LocalBlob/v1", map[string]interface{}{
				"mediaType": "application/vnd.oci.image.index.v1+json",
			}))
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Reference).To(Equal(repo + ":" + testResourceVersion + "@" + testDigest))
		})

		It("rejects a blob that is not an image manifest or index", func() {
			for _, mediaType := range []string{"application/vnd.oci.image.manifest.v1+tar+gzip", "application/octet-stream", ""} {
				_, err := template.ResolveImageReference(nil, localBlob("LocalBlob/v1", map[string]interface{}{"mediaType": mediaType}))
				Expect(err).To(MatchError(ContainSubstring("media type")), mediaType)
			}
		})

		It("fails for a localReference that is not a digest", func() {
			cv, _ := ociCV()
			_, err := template.ResolveImageReference(cv, localBlob("LocalBlob/v1", map[string]interface{}{"localReference": "latest"}))
			Expect(err).To(MatchError(ContainSubstring("localReference")))
		})

		It("fails without a localReference", func() {
			_, err := template.ResolveImageReference(nil, resource(testResourceVersion, map[string]interface{}{"type": "LocalBlob/v1", "mediaType": "application/vnd.oci.image.manifest.v1+json"}))
			Expect(err).To(MatchError(ContainSubstring("localReference")))
		})
	})

	It("reads the imageReference of any other access type", func() {
		ref, err := template.ResolveImageReference(nil, resource(testResourceVersion, map[string]interface{}{"type": "custom/v2", "imageReference": "ghcr.io/example/myimage:1.0.0"}))
		Expect(err).ToNot(HaveOccurred())
		Expect(ref.Repository).To(Equal("ghcr.io/example/myimage"))
	})

	It("fails for another access type without imageReference", func() {
		_, err := template.ResolveImageReference(nil, resource(testResourceVersion, map[string]interface{}{"type": "helm", "helmChart": "x"}))
		Expect(err).To(MatchError(ContainSubstring("imageReference")))
	})

	It("fails for a missing access", func() {
		_, err := template.ResolveImageReference(nil, resource(testResourceVersion, nil))
		Expect(err).To(MatchError(ContainSubstring("access")))
	})

	It("fails for a missing resource", func() {
		_, err := template.ResolveImageReference(nil, nil)
		Expect(err).To(MatchError(ContainSubstring("access")))
	})
})

// ocmContext returns an OCM context that is finalized when the spec ends.
func ocmContext() ocm.Context {
	octx := ocm.New(datacontext.MODE_EXTENDED)
	DeferCleanup(func() { Expect(octx.Finalize()).To(Succeed()) })
	return octx
}

// wrap turns an OCM library component version into the Landscaper model type.
func wrap(octx ocm.Context, ocmCV ocm.ComponentVersionAccess) model.ComponentVersion {
	factory := &ocmlib.Factory{}
	access, err := factory.NewRegistryAccess(octx.BindTo(context.Background()), &model.RegistryAccessOptions{})
	Expect(err).ToNot(HaveOccurred())
	cv, err := access.(*ocmlib.RegistryAccess).NewComponentVersion(ocmCV)
	Expect(err).ToNot(HaveOccurred())
	return cv
}

// newOCIComponentVersion stores a component version in an in-memory OCI registry under
// the sub path "internal". build may add resources before the version is written.
// It returns the component version and the registry host.
func newOCIComponentVersion(build func(ocm.ComponentVersionAccess)) (model.ComponentVersion, string) {
	server := httptest.NewServer(registry.New())
	DeferCleanup(server.Close)
	octx := ocmContext()

	repo, err := octx.RepositoryForSpec(ocireg.NewRepositorySpec(server.URL + "/internal"))
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(repo.Close)
	comp, err := repo.LookupComponent(testComponentName)
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(comp.Close)
	ocmCV, err := comp.NewVersion(testComponentVersion)
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(ocmCV.Close)
	if build != nil {
		build(ocmCV)
	}
	Expect(comp.AddVersion(ocmCV)).To(Succeed())

	return wrap(octx, ocmCV), strings.TrimPrefix(server.URL, "http://")
}

// newNonOCIComponentVersion returns a component version that lives in an in-memory
// composition repository, i.e. not in an OCI registry.
func newNonOCIComponentVersion() model.ComponentVersion {
	octx := ocmContext()
	repo := composition.NewRepository(octx, "test-repo")
	DeferCleanup(repo.Close)
	comp, err := repo.LookupComponent(testComponentName)
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(comp.Close)
	ocmCV, err := comp.NewVersion(testComponentVersion)
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(ocmCV.Close)
	return wrap(octx, ocmCV)
}
