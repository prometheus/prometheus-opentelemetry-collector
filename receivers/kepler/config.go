// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package kepler

import (
	"bytes"
	"fmt"

	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
	"gopkg.in/yaml.v3"
)

var _ prombridge.ConfigDecoder = configDecoder{}

// exporterConfig adapts Kepler's variadic validation API to the bridge hook.
type exporterConfig keplerconfig.Config

func (c *exporterConfig) Validate() error { return (*keplerconfig.Config)(c).ValidateRuntime() }

type configDecoder struct{}

// DecodeConfig preserves Kepler's native YAML keys and custom metrics-level
// decoder. Every call starts with fresh upstream defaults.
func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode Kepler config: %w", err)
	}
	cfg := keplerconfig.DefaultConfig()
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode Kepler config: %w", err)
	}
	return (*exporterConfig)(cfg), nil
}
