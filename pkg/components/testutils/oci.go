// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import cdv2 "github.com/openmcp-project/landscaper/legacy-component-spec/bindings-go/apis/v2"

func NewOCIRegistryAccess(ociImageRef string) (cdv2.UnstructuredTypedObject, error) {
	return cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(ociImageRef))
}

func NewOCIRepositoryContext(baseURL string) (cdv2.UnstructuredTypedObject, error) {
	return cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository(baseURL, ""))
}
