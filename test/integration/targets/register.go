// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"

	"github.com/openmcp-project/landscaper/controller-utils/pkg/logging"
	"github.com/openmcp-project/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	_, ctx := logging.FromContextOrNew(context.Background(), nil)

	TargetTests(f)
	TargetMapTests(ctx, f)
	OIDCTargetTests(ctx, f)
	SelfTargetTests(ctx, f)
}
