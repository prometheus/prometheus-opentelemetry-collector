// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import prombridge "github.com/prometheus/opentelemetry-collector-bridge"

var _ prombridge.ConfigUnmarshaler = configUnmarshaler{}

// exporterConfig is empty because ecs_exporter discovers the ECS task metadata
// endpoint from ECS_CONTAINER_METADATA_URI_V4.
type exporterConfig struct{}

type configUnmarshaler struct{}

func (configUnmarshaler) GetConfigStruct() any {
	return &exporterConfig{}
}
