// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"testing"
	"time"

	bbconfig "github.com/prometheus/blackbox_exporter/config"
)

func TestConfigDecoderStructuredModules(t *testing.T) {
	decoded, err := (configDecoder{}).DecodeConfig(map[string]interface{}{
		"modules": map[string]interface{}{
			"http_2xx": map[string]interface{}{
				"prober":  "http",
				"timeout": "5s",
			},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"name":    "example",
				"address": "https://example.com",
				"module":  "http_2xx",
			},
		},
		"probe_timeout_offset": "250ms",
		"max_timeout":          "10s",
	})
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	cfg := decoded.(*bbconfig.Config)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := cfg.Modules.Modules["http_2xx"].Timeout; got != 5*time.Second {
		t.Fatalf("module timeout = %v; want 5s", got)
	}
	if got := cfg.ProbeTimeoutOffset; got != 250*time.Millisecond {
		t.Fatalf("probe_timeout_offset = %v; want 250ms", got)
	}
}

func TestConfigDecoderRejectsUnknownField(t *testing.T) {
	_, err := (configDecoder{}).DecodeConfig(map[string]interface{}{
		"not_a_real_field": true,
	})
	if err == nil {
		t.Fatal("DecodeConfig() accepted an unknown field")
	}
}

func TestConfigRejectsModulesAndConfigFile(t *testing.T) {
	decoded, err := (configDecoder{}).DecodeConfig(map[string]interface{}{
		"modules": map[string]interface{}{
			"http_2xx": map[string]interface{}{"prober": "http"},
		},
		"config_file": "blackbox.yml",
		"targets": []interface{}{
			map[string]interface{}{
				"name":    "example",
				"address": "https://example.com",
				"module":  "http_2xx",
			},
		},
	})
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if err := decoded.(*bbconfig.Config).Validate(); err == nil {
		t.Fatal("Validate() accepted modules and config_file")
	}
}
