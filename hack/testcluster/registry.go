// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/spf13/cobra"
)

func NewRegistryCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry",
		Short:   "",
		Long:    "",
		Example: "",
	}
	cmd.AddCommand(NewCreateRegistryCommand(ctx))
	cmd.AddCommand(NewDeleteRegistryCommand(ctx))
	return cmd
}
