// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rootinstallations

import (
	"github.com/openmcp-project/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	RootInstallationTests(f)
}
