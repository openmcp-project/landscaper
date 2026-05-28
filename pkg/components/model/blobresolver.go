// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"io"

	"github.com/openmcp-project/landscaper/pkg/components/model/types"
)

type BlobResolver interface {
	Info(ctx context.Context, res types.Resource) (*types.BlobInfo, error)

	Resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error)
}
