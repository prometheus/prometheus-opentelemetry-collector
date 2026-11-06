// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package kepler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
)

func TestDecodeNativeKeplerConfig(t *testing.T) {
	raw := map[string]interface{}{
		"kube":     map[string]interface{}{"enabled": true, "nodeName": "worker-a", "podInformer": map[string]interface{}{"mode": "apiserver"}},
		"monitor":  map[string]interface{}{"interval": "2s"},
		"exporter": map[string]interface{}{"prometheus": map[string]interface{}{"metricsLevel": []interface{}{"node", "pod"}, "debugCollectors": []interface{}{}}},
	}
	decoded, err := (configDecoder{}).DecodeConfig(raw)
	require.NoError(t, err)
	cfg := decoded.(*exporterConfig)
	require.Equal(t, 2*time.Second, cfg.Monitor.Interval)
	require.Equal(t, keplerconfig.MetricsLevelNode|keplerconfig.MetricsLevelPod, cfg.Exporter.Prometheus.MetricsLevel)
	require.Equal(t, "worker-a", cfg.Kube.Node)
	require.Equal(t, "apiserver", cfg.Kube.PodInformer.Mode)
	cfg.Host.SysFS = "/"
	cfg.Host.ProcFS = "/"
	require.NoError(t, cfg.Validate())
}

func TestDecodeRejectsInvalidKeysAndTypes(t *testing.T) {
	for _, raw := range []map[string]interface{}{
		{"monitor": map[string]interface{}{"intervall": "2s"}},
		{"exporter": map[string]interface{}{"prometheus": map[string]interface{}{"metricsLevel": []interface{}{"unknown"}}}},
		{"kube": map[string]interface{}{"enabled": "not-a-bool"}},
	} {
		_, err := (configDecoder{}).DecodeConfig(raw)
		require.Error(t, err)
	}
}

func TestDecodedDefaultsAreIndependent(t *testing.T) {
	a, err := (configDecoder{}).DecodeConfig(nil)
	require.NoError(t, err)
	b, err := (configDecoder{}).DecodeConfig(nil)
	require.NoError(t, err)
	first, second := a.(*exporterConfig), b.(*exporterConfig)
	first.Cpu.PreferredMeters[0] = "fake"
	*first.Kube.Enabled = true
	require.Equal(t, "rapl", second.Cpu.PreferredMeters[0])
	require.False(t, *second.Kube.Enabled)
}
