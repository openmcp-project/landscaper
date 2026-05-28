// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "Types Testing")
}
