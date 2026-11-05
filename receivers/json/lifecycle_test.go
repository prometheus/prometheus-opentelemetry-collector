// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus-community/json_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/json/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestReceiverEmitsOTelMetrics(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/other" {
			_, _ = io.WriteString(w, `{"value":7}`)
			return
		}
		_, _ = io.WriteString(w, `{"value":42}`)
	}))
	defer target.Close()
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ScrapeInterval = 10 * time.Millisecond
	cfg.ExporterConfig = rawConfig(target.URL)
	cfg.ExporterConfig["targets"] = append(cfg.ExporterConfig["targets"].([]interface{}), map[string]interface{}{
		"name": "other", "address": target.URL + "/other",
	})
	sink := new(consumertest.MetricsSink)
	receiver, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, receiver.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, receiver.Shutdown(context.Background())) }()
	require.Eventually(t, func() bool { return sink.DataPointCount() >= 2 }, 5*time.Second, 10*time.Millisecond)
	found := false
	for _, batch := range sink.AllMetrics() {
		for _, rm := range batch.ResourceMetrics().All() {
			for _, sm := range rm.ScopeMetrics().All() {
				for _, metric := range sm.Metrics().All() {
					if metric.Name() != "api_value" {
						continue
					}
					found = true
					require.Equal(t, 2, metric.Gauge().DataPoints().Len())
					values := make(map[string]float64)
					for _, point := range metric.Gauge().DataPoints().All() {
						name, ok := point.Attributes().Get("target_name")
						require.True(t, ok)
						values[name.Str()] = point.DoubleValue()
					}
					require.Equal(t, map[string]float64{"api": 42, "other": 7}, values)
				}
			}
		}
	}
	require.True(t, found, "api_value missing from OTel pipeline")
}

func TestLifecycleInstancesAndPartialFailure(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `{"value":7}`)
	}))
	defer target.Close()
	cfg := decodeConfig(t, rawConfig(target.URL))
	cfg.Targets = append(cfg.Targets, config.Target{Name: "unavailable", Address: target.URL + "/fail", Module: "default"})
	manager := &lifecycleManager{}
	set := receivertest.NewNopSettings(metadata.Type)
	// The startup context must not become the runtime's lifetime context.
	ctx, cancel := context.WithCancel(context.Background())
	first, shutdownFirst, err := manager.Start(ctx, set, cfg)
	require.NoError(t, err)
	defer shutdownFirst(context.Background())
	second, shutdownSecond, err := manager.Start(ctx, set, cfg)
	require.NoError(t, err)
	defer shutdownSecond(context.Background())
	cancel()
	for _, registry := range []*prometheus.Registry{first, second} {
		families, err := registry.Gather()
		require.Error(t, err)
		require.Len(t, families, 1)
		require.Equal(t, "api_value", families[0].GetName())
		require.Equal(t, float64(7), families[0].Metric[0].GetGauge().GetValue())
	}
	require.NoError(t, shutdownFirst(context.Background()))
	_, err = first.Gather()
	require.Error(t, err)
	families, err := second.Gather()
	require.Error(t, err) // only the unavailable target fails
	require.Len(t, families, 1)
}

func TestLifecycleShutdownCancelsFetch(t *testing.T) {
	started := make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer target.Close()
	registry, shutdown, err := (&lifecycleManager{}).Start(context.Background(), receivertest.NewNopSettings(metadata.Type), decodeConfig(t, rawConfig(target.URL)))
	require.NoError(t, err)
	defer shutdown(context.Background())
	done := make(chan error, 1)
	go func() { _, err := registry.Gather(); done <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch never started")
	}
	require.NoError(t, shutdown(context.Background()))
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not cancel fetch")
	}
}
