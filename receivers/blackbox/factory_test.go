// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"testing"

	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/blackbox/internal/metadata"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	if NewFactory() == nil {
		t.Fatal("NewFactory() returned nil")
	}
}

func TestFactoryCreateMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ExporterConfig = map[string]interface{}{
		"modules": map[string]interface{}{
			"http_2xx": map[string]interface{}{"prober": "http"},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"name":    "example",
				"address": "https://example.com",
				"module":  "http_2xx",
			},
		},
	}
	_, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		new(consumertest.MetricsSink),
	)
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}
