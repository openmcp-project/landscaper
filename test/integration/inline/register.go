// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"github.com/openmcp-project/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	InlineBlueprintTests(f)
	InlineTemplateTests(f)
	ContextTests(f)
}
