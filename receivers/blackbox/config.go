// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/go-viper/mapstructure/v2"
	bbconfig "github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/blackbox_exporter/prober"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.yaml.in/yaml/v3"
)

const (
	defaultProbeTimeoutOffset = 500 * time.Millisecond
	defaultMaxTimeout         = 120 * time.Second
)

var _ prombridge.ConfigDecoder = configDecoder{}

type configDecoder struct{}

type Config struct {
	ConfigFile         string
	Modules            map[string]bbconfig.Module
	Targets            []Target
	ProbeTimeoutOffset time.Duration
	MaxTimeout         time.Duration
	hasInlineModules   bool
}

type Target struct {
	Name    string            `mapstructure:"name"`
	Address string            `mapstructure:"address"`
	Module  string            `mapstructure:"module"`
	Labels  map[string]string `mapstructure:"labels"`
}

type receiverExporterConfig struct {
	ConfigFile         string   `mapstructure:"config_file"`
	Targets            []Target `mapstructure:"targets"`
	ProbeTimeoutOffset string   `mapstructure:"probe_timeout_offset"`
	MaxTimeout         string   `mapstructure:"max_timeout"`
}

func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	remaining := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		remaining[key] = value
	}
	rawModules, hasModules := remaining["modules"]
	delete(remaining, "modules")

	wire := receiverExporterConfig{
		ProbeTimeoutOffset: defaultProbeTimeoutOffset.String(),
		MaxTimeout:         defaultMaxTimeout.String(),
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

	cfg := &Config{
		ConfigFile:       wire.ConfigFile,
		Targets:          wire.Targets,
		hasInlineModules: hasModules,
	}
	durationDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      cfg,
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

	if hasModules {
		data, err := yaml.Marshal(map[string]interface{}{"modules": rawModules})
		if err != nil {
			return nil, fmt.Errorf("marshal modules: %w", err)
		}
		var modules bbconfig.Config
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&modules); err != nil {
			return nil, fmt.Errorf("decode modules: %w", err)
		}
		cfg.Modules = modules.Modules
	} else if cfg.ConfigFile != "" {
		safeConfig := bbconfig.NewSafeConfig(prometheus.NewRegistry())
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		if err := safeConfig.ReloadConfig(cfg.ConfigFile, logger); err != nil {
			return nil, fmt.Errorf("load config_file: %w", err)
		}
		cfg.Modules = safeConfig.C.Modules
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.hasInlineModules && c.ConfigFile != "" {
		return fmt.Errorf("modules and config_file are mutually exclusive")
	}
	if len(c.Modules) == 0 {
		return fmt.Errorf("at least one module is required")
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	if c.ProbeTimeoutOffset < 0 {
		return fmt.Errorf("probe_timeout_offset cannot be negative")
	}
	if c.MaxTimeout <= 0 {
		return fmt.Errorf("max_timeout must be positive")
	}
	if c.ProbeTimeoutOffset >= c.MaxTimeout {
		return fmt.Errorf("probe_timeout_offset must be less than max_timeout")
	}
	for i, target := range c.Targets {
		if target.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if target.Address == "" {
			return fmt.Errorf("target %q: address is required", target.Name)
		}
		module, ok := c.Modules[target.Module]
		if !ok {
			return fmt.Errorf("target %q: unknown module %q", target.Name, target.Module)
		}
		if _, ok := prober.Probers[module.Prober]; !ok {
			return fmt.Errorf("module %q: unknown prober %q", target.Module, module.Prober)
		}
		for label := range target.Labels {
			switch label {
			case "target", "target_name", "module":
				return fmt.Errorf("target %q: label %q is reserved", target.Name, label)
			}
		}
	}
	return nil
}
