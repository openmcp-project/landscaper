// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	componentcliMetrics "github.com/openmcp-project/landscaper/legacy-component-cli/ociclient/metrics"
)

/*
  This package contains all metrics that are exposed by the landscaper.
  It offers a function to register the metrics on a prometheus registry
*/

// RegisterMetrics allows to register all landscaper exposed metrics
func RegisterMetrics(reg prometheus.Registerer) {
	componentcliMetrics.RegisterCacheMetrics(reg)
}
