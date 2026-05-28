// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/openmcp-project/landscaper/apis/config"
)

// ConvertCommonControllerConfigToControllerOptions converts the landscaper CommonControllerConfig to controller.Options.
func ConvertCommonControllerConfigToControllerOptions(cfg config.CommonControllerConfig) controller.Options {
	opts := controller.Options{
		MaxConcurrentReconciles: cfg.Workers,
	}
	if cfg.CacheSyncTimeout != nil {
		opts.CacheSyncTimeout = cfg.CacheSyncTimeout.Duration
	}
	return opts
}
