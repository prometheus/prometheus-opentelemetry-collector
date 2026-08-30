// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestLifecycleManagerEndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	decoded, err := (configDecoder{}).DecodeConfig(map[string]interface{}{
		"modules": map[string]interface{}{
			"http_2xx": map[string]interface{}{"prober": "http"},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"name":    "local",
				"address": server.URL,
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

	manager := newLifecycleManager()
	registry, err := manager.Start(
		context.Background(),
		receivertest.NewNopSettings(component.MustNewType("blackbox_exporter")),
		cfg,
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer manager.Shutdown(context.Background())

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	for _, family := range families {
		if family.GetName() == "probe_success" &&
			len(family.Metric) == 1 &&
			family.Metric[0].GetGauge().GetValue() == 1 {
			return
		}
	}
	t.Fatal("successful probe_success metric not found")
}

func TestLifecycleManagerRejectsWrongConfig(t *testing.T) {
	manager := newLifecycleManager()
	_, err := manager.Start(
		context.Background(),
		receivertest.NewNopSettings(component.MustNewType("blackbox_exporter")),
		struct{}{},
	)
	if err == nil {
		t.Fatal("Start() accepted the wrong config type")
	}
}
