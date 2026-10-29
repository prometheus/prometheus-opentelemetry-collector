// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package kepler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type runtime interface {
	Collectors() []prometheus.Collector
	Shutdown(context.Context) error
}

type lifecycleManager struct {
	newRuntime func(*keplerconfig.Config, *slog.Logger) (runtime, error)
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{newRuntime: newKeplerRuntime}
}

func (m *lifecycleManager) Start(ctx context.Context, set receiver.Settings, raw any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := raw.(*exporterConfig)
	if !ok {
		return nil, nil, fmt.Errorf("expected *exporterConfig, got %T", raw)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if set.Logger != nil {
		logger = slog.New(zapslog.NewHandler(set.Logger.Core()))
	}
	rt, err := m.newRuntime((*keplerconfig.Config)(cfg), logger)
	if err != nil {
		return nil, nil, fmt.Errorf("create Kepler runtime: %w", err)
	}
	registry := prometheus.NewRegistry()
	for _, c := range rt.Collectors() {
		if err := registry.Register(c); err != nil {
			return nil, nil, errors.Join(fmt.Errorf("register Kepler collector: %w", err), rt.Shutdown(ctx))
		}
	}
	return registry, rt.Shutdown, nil
}
