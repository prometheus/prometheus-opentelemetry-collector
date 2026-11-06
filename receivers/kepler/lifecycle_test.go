// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

//go:build linux

package kepler

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/kepler/internal/metadata"
	"github.com/stretchr/testify/require"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func fakeConfig(node string) *exporterConfig {
	cfg := keplerconfig.DefaultConfig()
	cfg.Cpu.PreferredMeters = []string{"fake"}
	cfg.Kube.Node = node
	cfg.Monitor.Interval = 10 * time.Millisecond
	cfg.Monitor.Staleness = 0
	cfg.Exporter.Prometheus.MetricsLevel = keplerconfig.MetricsLevelNode
	cfg.Exporter.Prometheus.DebugCollectors = nil
	return (*exporterConfig)(cfg)
}

func gatheredEnergy(t *testing.T, r *prometheus.Registry, node string) float64 {
	t.Helper()
	families, err := r.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() != "kepler_node_cpu_joules_total" {
			continue
		}
		for _, m := range f.Metric {
			for _, l := range m.Label {
				if l.GetName() == "node_name" {
					require.Equal(t, node, l.GetValue())
				}
			}
			return m.GetCounter().GetValue()
		}
	}
	return 0
}

func TestLifecycleIndependentRuntimes(t *testing.T) {
	manager := newLifecycleManager()
	ctx, cancel := context.WithCancel(context.Background())
	a, stopA, err := manager.Start(ctx, receivertest.NewNopSettings(metadata.Type), fakeConfig("node-a"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopA(context.Background())) })
	cancel() // startup cancellation must not stop background sampling
	b, stopB, err := manager.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), fakeConfig("node-b"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopB(context.Background())) })
	require.Eventually(t, func() bool { return gatheredEnergy(t, a, "node-a") > 0 && gatheredEnergy(t, b, "node-b") > 0 }, 3*time.Second, 10*time.Millisecond)
	before := gatheredEnergy(t, b, "node-b")
	require.NoError(t, stopA(context.Background()))
	require.NoError(t, stopA(context.Background()))
	require.Eventually(t, func() bool { return gatheredEnergy(t, b, "node-b") > before }, 3*time.Second, 10*time.Millisecond)
}

type fakeRuntime struct {
	collectors []prometheus.Collector
	shutdowns  atomic.Int32
}

func (r *fakeRuntime) Collectors() []prometheus.Collector { return r.collectors }
func (r *fakeRuntime) Shutdown(context.Context) error     { r.shutdowns.Add(1); return nil }

func TestLifecycleRegistrationFailureCleansRuntime(t *testing.T) {
	rt := &fakeRuntime{collectors: []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "duplicate"}),
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "duplicate"}),
	}}
	m := &lifecycleManager{newRuntime: func(*keplerconfig.Config, *slog.Logger) (runtime, error) { return rt, nil }}
	_, _, err := m.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), fakeConfig("test"))
	require.ErrorContains(t, err, "register Kepler collector")
	require.EqualValues(t, 1, rt.shutdowns.Load())
}

func TestLifecycleConstructionErrors(t *testing.T) {
	m := newLifecycleManager()
	_, _, err := m.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), struct{}{})
	require.ErrorContains(t, err, "expected *exporterConfig")
	failure := errors.New("hardware unavailable")
	m.newRuntime = func(*keplerconfig.Config, *slog.Logger) (runtime, error) { return nil, failure }
	_, _, err = m.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), fakeConfig("test"))
	require.ErrorIs(t, err, failure)
}

func TestFactoryDeliversMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ScrapeInterval = 10 * time.Millisecond
	cfg.ExporterConfig = map[string]interface{}{
		"cpu":      map[string]interface{}{"preferredMeters": []interface{}{"fake"}},
		"kube":     map[string]interface{}{"nodeName": "factory-test"},
		"monitor":  map[string]interface{}{"interval": "10ms", "staleness": "0s"},
		"exporter": map[string]interface{}{"prometheus": map[string]interface{}{"metricsLevel": []interface{}{"node"}, "debugCollectors": []interface{}{}}},
	}
	sink := new(consumertest.MetricsSink)
	receiver, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, receiver.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, receiver.Shutdown(context.Background())) })
	require.Eventually(t, func() bool {
		for _, batch := range sink.AllMetrics() {
			for _, rm := range batch.ResourceMetrics().All() {
				for _, sm := range rm.ScopeMetrics().All() {
					for _, metric := range sm.Metrics().All() {
						if metric.Name() == "kepler_node_cpu_joules_total" {
							return true
						}
					}
				}
			}
		}
		return false
	}, 3*time.Second, 10*time.Millisecond)
}
