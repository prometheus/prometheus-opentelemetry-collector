// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package node

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/node_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/node/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func receiverConfig(t *testing.T, factory receiver.Factory, exporter map[string]any) *prombridge.ReceiverConfig {
	t.Helper()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
		"scrape_interval": "25ms", "exporter_config": exporter,
	}).Unmarshal(cfg))
	return cfg
}

func TestFactoryDefaultsAndConfigIsolation(t *testing.T) {
	t.Parallel()
	factory := NewFactory()
	require.Equal(t, component.MustNewType("node_exporter"), factory.Type())
	first := receiverConfig(t, factory, map[string]any{
		"collector_disable_defaults": true,
		"collectors":                 map[string]any{"textfile": true},
		"cpu":                        map[string]any{"enable_guest": false},
		"path":                       map[string]any{"proc_fs": "custom/proc"},
		"filesystem":                 map[string]any{"mount_timeout": "2s"},
		"textfile":                   map[string]any{"directories": []any{"testdata/textfile-a"}},
	})
	second := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	require.Empty(t, second.ExporterConfig["collectors"])
	require.True(t, second.ExporterConfig["cpu"].(config.CPUConfig).EnableGuest)
	require.Equal(t, "/proc", second.ExporterConfig["path"].(config.PathConfig).ProcFS)
	require.Empty(t, second.ExporterConfig["textfile"].(config.TextfileConfig).Directories)
	require.Equal(t, "custom/proc", first.ExporterConfig["path"].(map[string]any)["proc_fs"])
	for _, cfg := range []*prombridge.ReceiverConfig{first, second} {
		_, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, new(consumertest.MetricsSink))
		require.NoError(t, err)
	}
}

func TestFactoryAcceptsIncludeWithDefaultExcludes(t *testing.T) {
	t.Parallel()
	factory := NewFactory()
	cfg := receiverConfig(t, factory, map[string]any{
		"collector_disable_defaults": true,
		"collectors":                 map[string]any{"textfile": true},
		"diskstats":                  map[string]any{"device_include": "^sd"},
		"filesystem":                 map[string]any{"mount_points_include": "^/data", "fs_types_include": "^ext4$"},
	})
	_, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, new(consumertest.MetricsSink))
	require.NoError(t, err)
}

func TestFactoryRejectsInvalidConfig(t *testing.T) {
	t.Parallel()
	for name, exporter := range map[string]map[string]any{
		"unknown field":       {"unknown_setting": true},
		"invalid regexp":      {"cpu": map[string]any{"flags_include": "("}},
		"invalid duration":    {"filesystem": map[string]any{"mount_timeout": "soon"}},
		"conflicting filters": {"filesystem": map[string]any{"mount_points_include": "^/data", "mount_points_exclude": "^/tmp"}},
	} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := receiverConfig(t, factory, exporter)
			_, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, new(consumertest.MetricsSink))
			require.Error(t, err)
		})
	}
}

func TestReceiverMultipleInstances(t *testing.T) {
	t.Parallel()
	factory := NewFactory()
	configs := make([]*prombridge.ReceiverConfig, 2)
	// Decode both configurations before constructing either receiver, as the
	// Collector does. Different fixture directories make cross-instance sharing observable.
	for i, directory := range []string{"textfile-a", "textfile-b"} {
		path, err := filepath.Abs(filepath.Join("testdata", directory))
		require.NoError(t, err)
		configs[i] = receiverConfig(t, factory, map[string]any{
			"collector_disable_defaults": true,
			"collectors":                 map[string]any{"textfile": true},
			"textfile":                   map[string]any{"directories": []any{path}},
		})
	}
	receivers := make([]receiver.Metrics, 2)
	sinks := make([]*consumertest.MetricsSink, 2)
	for i, cfg := range configs {
		set := receivertest.NewNopSettings(metadata.Type)
		set.ID = component.NewIDWithName(metadata.Type, fmt.Sprintf("instance%d", i))
		sinks[i] = new(consumertest.MetricsSink)
		r, err := factory.CreateMetrics(context.Background(), set, cfg, sinks[i])
		require.NoError(t, err)
		receivers[i] = r
		t.Cleanup(func() { require.NoError(t, r.Shutdown(context.Background())) })
		startup, cancel := context.WithCancel(context.Background())
		require.NoError(t, r.Start(startup, componenttest.NewNopHost()))
		cancel() // A startup context must not become the runtime's scrape context.
	}
	for i, sink := range sinks {
		require.Eventually(t, func() bool {
			return latestFixtureValue(sink, []float64{7, 11}[i])
		}, 5*time.Second, 10*time.Millisecond, "receiver %d must gather its own file", i)
	}
	require.NoError(t, receivers[0].Shutdown(context.Background()))
	require.NoError(t, receivers[0].Shutdown(context.Background()))
	stopped := len(sinks[0].AllMetrics())
	before := len(sinks[1].AllMetrics())
	require.Eventually(t, func() bool {
		return len(sinks[1].AllMetrics()) >= before+3 && latestFixtureValue(sinks[1], 11)
	}, 5*time.Second, 10*time.Millisecond)
	require.Len(t, sinks[0].AllMetrics(), stopped)
}

func latestFixtureValue(sink *consumertest.MetricsSink, want float64) bool {
	batches := sink.AllMetrics()
	if len(batches) == 0 {
		return false
	}
	rms := batches[len(batches)-1].ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		sms := rms.At(i).ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			metrics := sms.At(j).Metrics()
			for k := 0; k < metrics.Len(); k++ {
				m := metrics.At(k)
				if m.Name() != "node_embedding_fixture_value" || m.Type() != pmetric.MetricTypeGauge {
					continue
				}
				points := m.Gauge().DataPoints()
				for l := 0; l < points.Len(); l++ {
					point := points.At(l)
					if point.ValueType() == pmetric.NumberDataPointValueTypeDouble && point.DoubleValue() == want {
						return true
					}
					if point.ValueType() == pmetric.NumberDataPointValueTypeInt && float64(point.IntValue()) == want {
						return true
					}
				}
			}
		}
	}
	return false
}
