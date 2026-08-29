// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	bbconfig "github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/blackbox_exporter/prober"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type lifecycleManager struct {
	loggerFromSettings func(receiver.Settings) *slog.Logger
	runtime            *prober.Runtime
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{loggerFromSettings: collectorSlogLogger}
}

func (m *lifecycleManager) Start(_ context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, error) {
	cfg, ok := exporterCfg.(*bbconfig.Config)
	if !ok {
		return nil, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	runtime, err := prober.NewRuntime(*cfg, m.loggerFromSettings(set))
	if err != nil {
		return nil, fmt.Errorf("start blackbox runtime: %w", err)
	}
	registry := prometheus.NewRegistry()
	for _, collector := range runtime.Collectors() {
		if err := registry.Register(collector); err != nil {
			_ = runtime.Shutdown(context.Background())
			return nil, fmt.Errorf("register blackbox collector: %w", err)
		}
	}
	m.runtime = runtime
	return registry, nil
}

func (m *lifecycleManager) Shutdown(ctx context.Context) error {
	if m.runtime == nil {
		return nil
	}
	err := m.runtime.Shutdown(ctx)
	m.runtime = nil
	return err
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
