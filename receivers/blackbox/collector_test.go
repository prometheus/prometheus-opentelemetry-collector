// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bbconfig "github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestProbeCollectorSuccessfulProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := testConfig(server.URL, 0)
	cfg.Targets[0].Labels = map[string]string{"environment": "test"}
	families := gatherCollector(t, newProbeCollector(context.Background(), discardLogger(), cfg))

	metric := findMetric(t, families, "probe_success")
	if got := metric.GetGauge().GetValue(); got != 1 {
		t.Fatalf("probe_success = %v; want 1", got)
	}
	wantLabels := map[string]string{
		"environment": "test",
		"module":      "http_2xx",
		"target":      server.URL,
		"target_name": "local",
	}
	for _, pair := range metric.GetLabel() {
		if want, ok := wantLabels[pair.GetName()]; ok && want == pair.GetValue() {
			delete(wantLabels, pair.GetName())
		}
	}
	if len(wantLabels) != 0 {
		t.Fatalf("probe_success missing labels %v", wantLabels)
	}
}

func TestProbeCollectorFailedProbe(t *testing.T) {
	cfg := testConfig("http://127.0.0.1:0", 0)
	families := gatherCollector(t, newProbeCollector(context.Background(), discardLogger(), cfg))

	if got := findMetric(t, families, "probe_success").GetGauge().GetValue(); got != 0 {
		t.Fatalf("probe_success = %v; want 0", got)
	}
}

func TestProbeCollectorHonorsModuleTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(time.Second)
	}))
	defer server.Close()

	cfg := testConfig(server.URL, 50*time.Millisecond)
	start := time.Now()
	families := gatherCollector(t, newProbeCollector(context.Background(), discardLogger(), cfg))
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Gather() took %v; want less than 500ms", elapsed)
	}
	if got := findMetric(t, families, "probe_success").GetGauge().GetValue(); got != 0 {
		t.Fatalf("probe_success = %v; want 0", got)
	}
}

func testConfig(address string, timeout time.Duration) *Config {
	module := bbconfig.DefaultModule
	module.Prober = "http"
	module.Timeout = timeout
	return &Config{
		Modules: map[string]bbconfig.Module{"http_2xx": module},
		Targets: []Target{{
			Name:    "local",
			Address: address,
			Module:  "http_2xx",
		}},
		ProbeTimeoutOffset: 500 * time.Millisecond,
		MaxTimeout:         120 * time.Second,
	}
}

func gatherCollector(t *testing.T, collector prometheus.Collector) []*dto.MetricFamily {
	t.Helper()
	registry := prometheus.NewRegistry()
	if err := registry.Register(collector); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	return families
}

func findMetric(t *testing.T, families []*dto.MetricFamily, name string) *dto.Metric {
	t.Helper()
	for _, family := range families {
		if family.GetName() == name && len(family.Metric) > 0 {
			return family.Metric[0]
		}
	}
	t.Fatalf("metric %q not found", name)
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
