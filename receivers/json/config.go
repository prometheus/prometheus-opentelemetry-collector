// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-viper/mapstructure/v2"
	jsonconfig "github.com/prometheus-community/json_exporter/config"
	"github.com/prometheus/common/model"
	"go.yaml.in/yaml/v2"
)

type Config struct {
	ConfigFile       string
	Modules          map[string]jsonconfig.Module
	Targets          []Target
	Timeout          time.Duration
	hasInlineModules bool
}

type Target struct {
	Name       string            `mapstructure:"name"`
	Address    string            `mapstructure:"address"`
	Module     string            `mapstructure:"module"`
	Labels     map[string]string `mapstructure:"labels"`
	Parameters url.Values        `mapstructure:"parameters"`
}

type configDecoder struct{}

func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	remaining := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		remaining[k] = v
	}
	modules, inline := remaining["modules"]
	delete(remaining, "modules")
	wire := struct {
		ConfigFile string        `mapstructure:"config_file"`
		Targets    []Target      `mapstructure:"targets"`
		Timeout    time.Duration `mapstructure:"timeout"`
	}{Timeout: jsonconfig.NewRuntimeConfigWithDefaults().Timeout}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &wire, ErrorUnused: true, WeaklyTypedInput: false, TagName: "mapstructure",
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(remaining); err != nil {
		return nil, err
	}
	cfg := &Config{ConfigFile: wire.ConfigFile, Targets: wire.Targets, Timeout: wire.Timeout, hasInlineModules: inline}
	if inline && wire.ConfigFile != "" {
		return nil, fmt.Errorf("modules and config_file are mutually exclusive")
	}
	var exporterCfg jsonconfig.Config
	if inline {
		data, err := yaml.Marshal(map[string]interface{}{"modules": modules})
		if err != nil {
			return nil, err
		}
		// Use the exporter's YAML types, including Prometheus authentication and TLS defaults.
		if err := yaml.UnmarshalStrict(data, &exporterCfg); err != nil {
			return nil, fmt.Errorf("decode modules: %w", err)
		}
	} else if wire.ConfigFile != "" {
		var err error
		exporterCfg, err = jsonconfig.LoadConfig(wire.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("load config_file: %w", err)
		}
	}
	cfg.Modules = exporterCfg.Modules
	for name, module := range cfg.Modules {
		cfg.Modules[name] = module.WithDefaults()
	}
	for i := range cfg.Targets {
		if cfg.Targets[i].Module == "" {
			cfg.Targets[i].Module = "default"
		}
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
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	names := make(map[string]bool, len(c.Targets))
	for _, target := range c.Targets {
		if target.Name == "" {
			return fmt.Errorf("target name is required")
		}
		if names[target.Name] {
			return fmt.Errorf("duplicate target name %q", target.Name)
		}
		names[target.Name] = true
		module, ok := c.Modules[target.Module]
		if !ok {
			return fmt.Errorf("target %q: unknown module %q", target.Name, target.Module)
		}
		if err := c.runtimeConfig(target).Validate(); err != nil {
			return fmt.Errorf("target %q: %w", target.Name, err)
		}
		for key := range target.Labels {
			if !model.LabelName(key).IsValid() {
				return fmt.Errorf("target %q: invalid label %q", target.Name, key)
			}
			switch key {
			case "target", "target_name", "module":
				return fmt.Errorf("target %q: label %q is reserved", target.Name, key)
			}
		}
		for _, metric := range module.Metrics {
			for key := range metric.Labels {
				switch key {
				case "target", "target_name", "module":
					return fmt.Errorf("module %q: metric label %q is reserved", target.Module, key)
				}
				if _, ok := target.Labels[key]; ok {
					return fmt.Errorf("target %q: label %q collides with a metric label", target.Name, key)
				}
			}
		}
	}
	return nil
}

func (c *Config) runtimeConfig(target Target) jsonconfig.RuntimeConfig {
	params := make(url.Values, len(target.Parameters)+2)
	for k, v := range target.Parameters {
		params[k] = append([]string(nil), v...)
	}
	params.Set("target", target.Address)
	params.Set("module", target.Module)
	return jsonconfig.RuntimeConfig{Target: target.Address, Module: c.Modules[target.Module], Parameters: params, Timeout: c.Timeout}
}
