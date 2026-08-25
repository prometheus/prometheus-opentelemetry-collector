// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	bbconfig "github.com/prometheus/blackbox_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/blackbox/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	defaults := bbconfig.NewConfigWithDefaults()
	return prombridge.NewFactoryWithDecoder(
		metadata.Type,
		newLifecycleManager(),
		configDecoder{},
		prombridge.WithComponentDefaults(map[string]interface{}{
			"probe_timeout_offset": defaults.ProbeTimeoutOffset.String(),
			"max_timeout":          defaults.MaxTimeout.String(),
		}),
	)
}
