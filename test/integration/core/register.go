// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/openmcp-project/landscaper/test/framework"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	RegistryTest(f)
}
