// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"github.com/openmcp-project/landscaper/pkg/components/model"
	"github.com/openmcp-project/landscaper/pkg/components/ocmlib"
)

var (
	ocmFactory model.Factory = &ocmlib.Factory{}
)

func GetFactory() model.Factory {
	return ocmFactory
}
