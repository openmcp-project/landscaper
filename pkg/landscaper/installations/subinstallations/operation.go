// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import "github.com/openmcp-project/landscaper/pkg/landscaper/installations"

// Operation contains all subinstallation operations
type Operation struct {
	*installations.Operation
}

// New creates a new subinstallation operation
func New(op *installations.Operation) *Operation {
	return &Operation{
		Operation: op,
	}
}
