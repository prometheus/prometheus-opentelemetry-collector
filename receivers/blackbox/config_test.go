// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"os"
	"path/filepath"
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
	cfg := decoded.(*Config)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := cfg.Modules["http_2xx"].Timeout; got != 5*time.Second {
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
	if err := decoded.(*Config).Validate(); err == nil {
		t.Fatal("Validate() accepted modules and config_file")
	}
}

func TestConfigDecoderLoadsConfigFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "blackbox.yml")
	if err := os.WriteFile(configFile, []byte(`
modules:
  http_2xx:
    prober: http
    timeout: 3s
`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	decoded, err := (configDecoder{}).DecodeConfig(map[string]interface{}{
		"config_file": configFile,
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
	cfg := decoded.(*Config)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := cfg.Modules["http_2xx"].Timeout; got != 3*time.Second {
		t.Fatalf("module timeout = %v; want 3s", got)
	}
}

func TestConfigRejectsUnknownModule(t *testing.T) {
	cfg := &Config{
		Modules: map[string]bbconfig.Module{
			"http_2xx": {Prober: "http"},
		},
		Targets: []Target{{Name: "example", Address: "https://example.com", Module: "missing"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted a target with an unknown module")
	}
}

func TestConfigRejectsUnknownProber(t *testing.T) {
	cfg := &Config{
		Modules: map[string]bbconfig.Module{
			"invalid": {Prober: "not-a-prober"},
		},
		Targets: []Target{{Name: "example", Address: "https://example.com", Module: "invalid"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted an unknown prober")
	}
}
