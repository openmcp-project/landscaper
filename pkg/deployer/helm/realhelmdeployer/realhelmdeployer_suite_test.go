// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRealHelmDeployer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RealHelmDeployer Test Suite")
}
