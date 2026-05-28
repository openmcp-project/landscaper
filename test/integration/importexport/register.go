// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"github.com/openmcp-project/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	ImportExportTests(f)
	ImportDataMappingsTests(f)
	ImportValidationTests(f)
	ImportExecutionsTests(f)
}
