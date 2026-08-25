// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	bbconfig "github.com/prometheus/blackbox_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.yaml.in/yaml/v3"
)

var _ prombridge.ConfigDecoder = configDecoder{}

type configDecoder struct{}

type receiverExporterConfig struct {
	ConfigFile         string            `mapstructure:"config_file"`
	Targets            []bbconfig.Target `mapstructure:"targets"`
	ProbeTimeoutOffset string            `mapstructure:"probe_timeout_offset"`
	MaxTimeout         string            `mapstructure:"max_timeout"`
}

func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	cfg := bbconfig.NewConfigWithDefaults()
	remaining := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		remaining[key] = value
	}
	rawModules, hasModules := remaining["modules"]
	delete(remaining, "modules")

	wire := receiverExporterConfig{
		ProbeTimeoutOffset: cfg.ProbeTimeoutOffset.String(),
		MaxTimeout:         cfg.MaxTimeout.String(),
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &wire,
		ErrorUnused:      true,
		WeaklyTypedInput: false,
		TagName:          "mapstructure",
	})
	if err != nil {
		return nil, fmt.Errorf("create config decoder: %w", err)
	}
	if err := decoder.Decode(remaining); err != nil {
		return nil, err
	}

	durationDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &cfg,
		ErrorUnused: true,
		TagName:     "mapstructure",
		DecodeHook:  mapstructure.StringToTimeDurationHookFunc(),
	})
	if err != nil {
		return nil, fmt.Errorf("create duration decoder: %w", err)
	}
	if err := durationDecoder.Decode(map[string]interface{}{
		"ProbeTimeoutOffset": wire.ProbeTimeoutOffset,
		"MaxTimeout":         wire.MaxTimeout,
	}); err != nil {
		return nil, err
	}
	cfg.ConfigFile = wire.ConfigFile
	cfg.Targets = wire.Targets

	if hasModules {
		data, err := yaml.Marshal(map[string]interface{}{"modules": rawModules})
		if err != nil {
			return nil, fmt.Errorf("marshal modules: %w", err)
		}
		modules, err := bbconfig.Load(data)
		if err != nil {
			return nil, fmt.Errorf("decode modules: %w", err)
		}
		cfg.Modules = *modules
	}
	return &cfg, nil
}
