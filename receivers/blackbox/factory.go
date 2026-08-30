// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/blackbox/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return prombridge.NewFactoryWithDecoder(
		metadata.Type,
		newLifecycleManager(),
		configDecoder{},
		prombridge.WithComponentDefaults(map[string]interface{}{
			"probe_timeout_offset": defaultProbeTimeoutOffset.String(),
			"max_timeout":          defaultMaxTimeout.String(),
		}),
	)
}
