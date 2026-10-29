// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus-community/json_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type lifecycleManager struct{}

func (*lifecycleManager) Start(_ context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := exporterCfg.(*Config)
	if !ok {
		return nil, nil, fmt.Errorf("expected *json.Config, got %T", exporterCfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if set.Logger != nil {
		logger = slog.New(zapslog.NewHandler(set.Logger.Core()))
	}
	// The exporter owns all target collectors and their lifecycle. This adapter
	// only registers them with the bridge and passes through instance shutdown.
	runtime, err := exporter.NewRuntime(cfg.Config, logger)
	if err != nil {
		return nil, nil, err
	}
	registry := prometheus.NewRegistry()
	for _, collector := range runtime.Collectors() {
		if err := registry.Register(collector); err != nil {
			_ = runtime.Shutdown(context.Background())
			return nil, nil, fmt.Errorf("register JSON collector: %w", err)
		}
	}
	return registry, runtime.Shutdown, nil
}
