// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package mysqld

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/mysqld/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// Decode all configurations before creating receivers, as the Collector does.
// Shared bridge defaults used to make every instance use the last one's DSN.
func multiInstanceConfigs(t *testing.T, factory receiver.Factory, dsns []string) []*prombridge.ReceiverConfig {
	t.Helper()
	configs := make([]*prombridge.ReceiverConfig, len(dsns))
	for i := range configs {
		configs[i] = factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	}
	for i, dsn := range dsns {
		fields := map[string]any{
			"scrape_interval": "100ms",
			"exporter_config": map[string]any{
				"data_source_name":       dsn,
				"exporter_query_timeout": "2s",
			},
		}
		if i == 0 {
			exporter := fields["exporter_config"].(map[string]any)
			exporter["collectors"] = map[string]any{"slave_status": false}
			exporter["heartbeat"] = map[string]any{"database": "custom_heartbeat"}
		}
		require.NoError(t, confmap.NewFromStringMap(fields).Unmarshal(configs[i]))
	}
	return configs
}

func TestFactory_MultipleInstanceConfigs(t *testing.T) {
	t.Parallel()
	factory := NewFactory()
	dsns := []string{"first@tcp(first:3306)/", "second@tcp(second:3306)/", "third@tcp(third:3306)/", "fourth@tcp(fourth:3306)/"}
	configs := multiInstanceConfigs(t, factory, dsns)
	for i, cfg := range configs {
		require.Equal(t, dsns[i], cfg.ExporterConfig["data_source_name"])
		want := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
		if i > 0 {
			require.Equal(t, want.ExporterConfig["collectors"], cfg.ExporterConfig["collectors"])
			require.Equal(t, want.ExporterConfig["heartbeat"], cfg.ExporterConfig["heartbeat"])
		}
		_, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, new(consumertest.MetricsSink))
		require.NoError(t, err)
	}
	fresh := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	require.Empty(t, fresh.ExporterConfig["data_source_name"])
}

// Run against two MySQL servers with different @@server_id values:
// TEST_MYSQL_DSN_A='root@tcp(127.0.0.1:3306)/' TEST_MYSQL_DSN_B='root@tcp(127.0.0.1:3307)/' go test -race -run TestIntegration ./...
func TestIntegration_MultipleInstances(t *testing.T) {
	a, b := os.Getenv("TEST_MYSQL_DSN_A"), os.Getenv("TEST_MYSQL_DSN_B")
	if a == "" || b == "" {
		t.Skip("set TEST_MYSQL_DSN_A and TEST_MYSQL_DSN_B to two MySQL servers with distinct server IDs")
	}
	serverID := func(dsn string) float64 {
		t.Helper()
		db, err := sql.Open("mysql", dsn)
		require.NoError(t, err)
		defer db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var id float64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT @@server_id").Scan(&id))
		return id
	}
	ids := []float64{serverID(a), serverID(b)}
	require.NotEqual(t, ids[0], ids[1], "servers must have distinct IDs to detect DSN contamination")
	factory := NewFactory()
	configs := multiInstanceConfigs(t, factory, []string{a, b, a, b})
	receivers := make([]receiver.Metrics, len(configs))
	sinks := make([]*consumertest.MetricsSink, len(configs))
	for i, cfg := range configs {
		set := receivertest.NewNopSettings(metadata.Type)
		set.ID = component.NewIDWithName(metadata.Type, fmt.Sprintf("instance%d", i))
		sinks[i] = new(consumertest.MetricsSink)
		r, err := factory.CreateMetrics(context.Background(), set, cfg, sinks[i])
		require.NoError(t, err)
		receivers[i] = r
		t.Cleanup(func() { require.NoError(t, r.Shutdown(context.Background())) })
		require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	}
	for i, sink := range sinks {
		require.Eventually(t, func() bool { return latestMySQLMetrics(sink, ids[i%2]) }, 15*time.Second, 50*time.Millisecond, "instance %d must scrape its own server", i)
	}
	// Stop the first instance after all four have started. The old manager shut
	// down the last runtime instead, so that receiver's subsequent mysql_up was 0.
	require.NoError(t, receivers[0].Shutdown(context.Background()))
	require.NoError(t, receivers[0].Shutdown(context.Background()))
	stoppedCount := len(sinks[0].AllMetrics())
	for i := 1; i < len(receivers); i++ {
		before := len(sinks[i].AllMetrics())
		require.Eventually(t, func() bool {
			return len(sinks[i].AllMetrics()) >= before+3 && latestMySQLMetrics(sinks[i], ids[i%2])
		}, 15*time.Second, 50*time.Millisecond, "instance %d must stay healthy after instance 0 shuts down", i)
	}
	require.Len(t, sinks[0].AllMetrics(), stoppedCount, "stopped receiver must stop exporting")
}

func latestMySQLMetrics(sink *consumertest.MetricsSink, serverID float64) bool {
	batches := sink.AllMetrics()
	if len(batches) == 0 {
		return false
	}
	var up, server bool
	rms := batches[len(batches)-1].ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		sms := rms.At(i).ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			metrics := sms.At(j).Metrics()
			for k := 0; k < metrics.Len(); k++ {
				m := metrics.At(k)
				if m.Type() != pmetric.MetricTypeGauge {
					continue
				}
				points := m.Gauge().DataPoints()
				for l := 0; l < points.Len(); l++ {
					p := points.At(l)
					value := p.DoubleValue()
					if p.ValueType() == pmetric.NumberDataPointValueTypeInt {
						value = float64(p.IntValue())
					}
					if m.Name() == "mysql_up" && value == 1 {
						up = true
					}
					if m.Name() == "mysql_global_variables_server_id" && value == serverID {
						server = true
					}
				}
			}
		}
	}
	return up && server
}
