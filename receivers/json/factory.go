// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/json/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory creates a receiver backed by the embedded JSON exporter.
func NewFactory() receiver.Factory {
	return prombridge.NewFactoryWithDecoder(metadata.Type, &lifecycleManager{}, configDecoder{})
}
