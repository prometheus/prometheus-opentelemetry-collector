// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func rawConfig(address string) map[string]interface{} {
	return map[string]interface{}{
		"modules": map[string]interface{}{"default": map[string]interface{}{
			"http_client_config": map[string]interface{}{},
			"metrics":            []interface{}{map[string]interface{}{"name": "api_value", "path": "{.value}", "valuetype": "gauge"}},
		}},
		"targets": []interface{}{map[string]interface{}{"name": "api", "address": address}},
	}
}

func decodeConfig(t *testing.T, raw map[string]interface{}) *Config {
	t.Helper()
	decoded, err := (configDecoder{}).DecodeConfig(raw)
	require.NoError(t, err)
	cfg := decoded.(*Config)
	require.NoError(t, cfg.Validate())
	return cfg
}

func TestConfigInlineAndFile(t *testing.T) {
	cfg := decodeConfig(t, rawConfig("https://example.com/api"))
	require.Equal(t, 30*time.Second, cfg.Timeout)
	require.Equal(t, "default", cfg.Targets[0].Module)
	metric := cfg.Modules["default"].Metrics[0]
	require.Equal(t, "api_value", metric.Help)
	require.Equal(t, "value", string(metric.Type))
	require.True(t, cfg.Modules["default"].HTTPClientConfig.FollowRedirects)
	dir := t.TempDir()
	file := filepath.Join(dir, "json.yml")
	require.NoError(t, os.WriteFile(file, []byte("modules:\n  default:\n    http_client_config: {}\n    metrics:\n      - name: api_value\n        path: '{.value}'\n        valuetype: gauge\n"), 0600))
	raw := rawConfig("https://example.com/api")
	delete(raw, "modules")
	raw["config_file"] = file
	fromFile := decodeConfig(t, raw)
	require.Equal(t, cfg.Modules, fromFile.Modules)
}

func TestConfigRejectsInvalidInputs(t *testing.T) {
	cases := map[string]func(map[string]interface{}){
		"unknown field": func(r map[string]interface{}) { r["typo"] = true },
		"unknown module field": func(r map[string]interface{}) {
			r["modules"].(map[string]interface{})["default"].(map[string]interface{})["typo"] = true
		},
		"unknown target field": func(r map[string]interface{}) {
			r["targets"].([]interface{})[0].(map[string]interface{})["typo"] = true
		},
		"file and modules": func(r map[string]interface{}) { r["config_file"] = "missing.yml" },
		"unknown module": func(r map[string]interface{}) {
			r["targets"].([]interface{})[0].(map[string]interface{})["module"] = "missing"
		},
		"bad target": func(r map[string]interface{}) {
			r["targets"].([]interface{})[0].(map[string]interface{})["address"] = "file:///tmp/test"
		},
		"no targets":   func(r map[string]interface{}) { delete(r, "targets") },
		"no modules":   func(r map[string]interface{}) { delete(r, "modules") },
		"zero timeout": func(r map[string]interface{}) { r["timeout"] = "0s" },
		"reserved label": func(r map[string]interface{}) {
			r["targets"].([]interface{})[0].(map[string]interface{})["labels"] = map[string]interface{}{"module": "bad"}
		},
		"duplicate name": func(r map[string]interface{}) {
			r["targets"] = append(r["targets"].([]interface{}), r["targets"].([]interface{})[0])
		},
		"metric label collision": func(r map[string]interface{}) {
			r["modules"].(map[string]interface{})["default"].(map[string]interface{})["metrics"].([]interface{})[0].(map[string]interface{})["labels"] = map[string]interface{}{"module": "bad"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			raw := rawConfig("https://example.com")
			mutate(raw)
			decoded, err := (configDecoder{}).DecodeConfig(raw)
			if err == nil {
				err = decoded.(*Config).Validate()
			}
			require.Error(t, err)
		})
	}
}
