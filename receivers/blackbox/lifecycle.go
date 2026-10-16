// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type lifecycleManager struct {
	loggerFromSettings func(receiver.Settings) *slog.Logger
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{loggerFromSettings: collectorSlogLogger}
}

func (m *lifecycleManager) Start(_ context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := exporterCfg.(*Config)
	if !ok {
		return nil, nil, fmt.Errorf("expected *blackbox.Config, got %T", exporterCfg)
	}

	registry := prometheus.NewRegistry()
	probeCtx, cancel := context.WithCancel(context.Background())
	if err := registry.Register(newProbeCollector(probeCtx, m.loggerFromSettings(set), cfg)); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("register blackbox probe collector: %w", err)
	}
	return registry, func(context.Context) error {
		cancel()
		return nil
	}, nil
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
