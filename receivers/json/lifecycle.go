// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package json

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	jsonexporter "github.com/prometheus-community/json_exporter/exporter"
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
	registry := prometheus.NewRegistry()
	runtimes := make([]*jsonexporter.Runtime, 0, len(cfg.Targets))
	shutdown := func(ctx context.Context) error {
		var err error
		for _, runtime := range runtimes {
			err = errors.Join(err, runtime.Shutdown(ctx))
		}
		return err
	}
	for _, target := range cfg.Targets {
		runtime, err := jsonexporter.NewRuntime(cfg.runtimeConfig(target), logger.With("target_name", target.Name, "module", target.Module))
		if err != nil {
			_ = shutdown(context.Background())
			return nil, nil, fmt.Errorf("target %q: %w", target.Name, err)
		}
		runtimes = append(runtimes, runtime)
		labels := prometheus.Labels{"target": target.Address, "target_name": target.Name, "module": target.Module}
		for k, v := range target.Labels {
			labels[k] = v
		}
		registerer := prometheus.WrapRegistererWith(labels, registry)
		for _, collector := range runtime.Collectors() {
			if err := registerer.Register(collector); err != nil {
				_ = shutdown(context.Background())
				return nil, nil, fmt.Errorf("register target %q: %w", target.Name, err)
			}
		}
	}
	return registry, shutdown, nil
}
