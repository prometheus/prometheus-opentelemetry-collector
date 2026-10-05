// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/ecs/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type capturingLifecycleManager struct {
	cfg any
}

func (m *capturingLifecycleManager) Start(_ context.Context, _ receiver.Settings, cfg any) (*prometheus.Registry, error) {
	m.cfg = cfg
	return prometheus.NewRegistry(), nil
}

func (*capturingLifecycleManager) Shutdown(context.Context) error { return nil }

func TestNewFactory(t *testing.T) {
	t.Parallel()

	if NewFactory() == nil {
		t.Fatal("NewFactory() returned nil")
	}
}

func TestFactoryCreateDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)
	if len(cfg.ExporterConfig) != 0 {
		t.Fatalf("ExporterConfig = %#v, want empty", cfg.ExporterConfig)
	}
}

func TestFactoryCreateMetricsDecodesConfig(t *testing.T) {
	t.Parallel()

	lifecycleManager := &capturingLifecycleManager{}
	factory := newFactoryWithLifecycleManager(lifecycleManager)
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	recv, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		new(consumertest.MetricsSink),
	)
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
	if err := recv.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := recv.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	if _, ok := lifecycleManager.cfg.(*exporterConfig); !ok {
		t.Fatalf("lifecycle received unexpected config type %T", lifecycleManager.cfg)
	}
}

func TestFactoryType(t *testing.T) {
	t.Parallel()

	if got, want := NewFactory().Type(), component.MustNewType("ecs_exporter"); got != want {
		t.Fatalf("factory type = %v, want %v", got, want)
	}
}
