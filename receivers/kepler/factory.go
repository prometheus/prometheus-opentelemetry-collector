// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package kepler

import (
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/kepler/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory creates a receiver that embeds Kepler's energy monitoring runtime.
func NewFactory() receiver.Factory {
	return prombridge.NewFactoryWithDecoder(metadata.Type, newLifecycleManager(), configDecoder{})
}
