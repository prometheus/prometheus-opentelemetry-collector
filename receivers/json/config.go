// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"fmt"
	"maps"

	"github.com/prometheus-community/json_exporter/config"
	"go.yaml.in/yaml/v2"
)

// Config uses the exporter's configuration model, with the additional requirement
// that a scheduled receiver has named targets and bounded requests.
type Config struct {
	config.Config
}

type configDecoder struct{}

func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	remaining := maps.Clone(raw)
	file := ""
	if value, exists := remaining["config_file"]; exists {
		var ok bool
		file, ok = value.(string)
		if !ok {
			return nil, fmt.Errorf("config_file must be a string")
		}
		delete(remaining, "config_file")
	}
	cfg := config.NewConfigWithDefaults()
	if file != "" {
		if _, exists := remaining["modules"]; exists {
			return nil, fmt.Errorf("modules and config_file are mutually exclusive")
		}
		var err error
		cfg, err = config.LoadConfig(file)
		if err != nil {
			return nil, fmt.Errorf("load config_file: %w", err)
		}
	}
	data, err := yaml.Marshal(remaining)
	if err != nil {
		return nil, fmt.Errorf("marshal exporter config: %w", err)
	}
	// Decode native exporter YAML directly. When loading a file, explicitly supplied
	// targets and timeout override its settings; omitted fields retain file values.
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode exporter config: %w", err)
	}
	return &Config{Config: cfg.WithDefaults()}, nil
}

func (c *Config) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	for _, target := range c.Targets {
		if target.Name == "" {
			return fmt.Errorf("target name is required")
		}
	}
	return nil
}
