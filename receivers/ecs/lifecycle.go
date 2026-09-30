// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/prometheus-community/ecs_exporter/ecscollector"
	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type runtime interface {
	Collectors() []prometheus.Collector
	Shutdown(context.Context) error
}

type lifecycleManager struct {
	loggerFromSettings       func(receiver.Settings) *slog.Logger
	newClientFromEnvironment func() (*ecsmetadata.Client, error)
	newRuntime               func(*ecsmetadata.Client, *slog.Logger) runtime

	mu      sync.Mutex
	runtime runtime
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{
		loggerFromSettings:       collectorSlogLogger,
		newClientFromEnvironment: ecsmetadata.NewClientFromEnvironment,
		newRuntime: func(client *ecsmetadata.Client, logger *slog.Logger) runtime {
			return ecscollector.NewRuntime(client, logger)
		},
	}
}

func (m *lifecycleManager) Start(ctx context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, error) {
	if _, ok := exporterCfg.(*exporterConfig); !ok {
		return nil, fmt.Errorf("expected *exporterConfig, got %T", exporterCfg)
	}

	client, err := m.newClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("create ECS metadata client: %w", err)
	}
	runtime := m.newRuntime(client, m.loggerFromSettings(set))

	registry := prometheus.NewRegistry()
	for _, collector := range runtime.Collectors() {
		if err := registry.Register(collector); err != nil {
			if shutdownErr := runtime.Shutdown(ctx); shutdownErr != nil {
				return nil, fmt.Errorf("register collector: %w; shutdown ECS runtime: %w", err, shutdownErr)
			}
			return nil, fmt.Errorf("register collector: %w", err)
		}
	}

	m.mu.Lock()
	m.runtime = runtime
	m.mu.Unlock()
	return registry, nil
}

func (m *lifecycleManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	runtime := m.runtime
	m.runtime = nil
	m.mu.Unlock()

	if runtime == nil {
		return nil
	}
	return runtime.Shutdown(ctx)
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
