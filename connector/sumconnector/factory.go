// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package sumconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/sumconnector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/sumconnector/internal/metadata"
)

// NewFactory returns a ConnectorFactory.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		connector.WithTracesToMetrics(nil, metadata.TracesToMetricsStability),
		connector.WithMetricsToMetrics(nil, metadata.MetricsToMetricsStability),
		connector.WithLogsToMetrics(nil, metadata.LogsToMetricsStability),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &Config{}
}

