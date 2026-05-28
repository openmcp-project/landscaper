// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"time"

	"github.com/openmcp-project/landscaper/test/framework"
)

var (
	resyncTime  = 1 * time.Second
	timeoutTime = 30 * time.Second
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	GenerationHandlingTestsForNewReconcile(f)
}
