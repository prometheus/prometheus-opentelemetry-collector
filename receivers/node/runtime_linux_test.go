// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package node

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/node_exporter/config"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/node/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestLinuxIndependentCollectorsAndPaths(t *testing.T) {
	t.Parallel()
	manager := newLifecycleManager()
	settings := receivertest.NewNopSettings(metadata.Type)
	tests := []struct {
		name, proc, metric string
		count              int
	}{
		{"cpu", "proc-a", "node_cpu_seconds_total", 8},
		{"meminfo", "proc-b", "node_memory_MemTotal_bytes", 1},
	}
	registries := make([]*prometheus.Registry, len(tests))
	for i, test := range tests {
		cfg := config.NewConfigWithDefaults()
		cfg.CollectorDisableDefaults = true
		cfg.Collectors = map[string]bool{test.name: true}
		cfg.Path.ProcFS = filepath.Join("testdata", test.proc)
		cfg.Path.SysFS = filepath.Join("testdata", "sys")
		registry, shutdown, err := manager.Start(context.Background(), settings, &cfg)
		require.NoError(t, err)
		require.Nil(t, shutdown, "upstream currently has no shutdown API")
		registries[i] = registry
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := registries[i]
			count, err := testutil.GatherAndCount(registry, test.metric)
			require.NoError(t, err)
			require.Equal(t, test.count, count)
			// The config passed into the constructor is complete; no kingpin parsing
			// or process-global path initialization is performed by this receiver.
			families, err := registry.Gather()
			require.NoError(t, err)
			for _, family := range families {
				if family.GetName() == "node_scrape_collector_success" {
					require.Len(t, family.Metric, 1)
					require.Equal(t, test.name, family.Metric[0].Label[0].GetValue())
					require.Equal(t, 1.0, family.Metric[0].Gauge.GetValue())
				}
			}
		})
	}
}
