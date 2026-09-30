// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import (
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/ecs/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory creates an ECS exporter receiver factory.
func NewFactory() receiver.Factory {
	return newFactoryWithLifecycleManager(newLifecycleManager())
}

func newFactoryWithLifecycleManager(lifecycleManager prombridge.ExporterLifecycleManager) receiver.Factory {
	return prombridge.NewFactoryWithUntaggedConfig(
		metadata.Type,
		lifecycleManager,
		configUnmarshaler{},
	)
}
