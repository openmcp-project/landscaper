// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"

	"github.com/openmcp-project/landscaper/test/utils/envtest"
)

type CleanupFunc func(ctx context.Context) error

// State wraps the envtest.State with framework related functionality.
type State struct {
	*envtest.State
	dumper  *Dumper
	cleanup CleanupFunc
}
