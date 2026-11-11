// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package node

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	nodecollector "github.com/prometheus/node_exporter/collector"
	"github.com/prometheus/node_exporter/config"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/node/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestLifecycleRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	manager := newLifecycleManager()
	settings := receivertest.NewNopSettings(metadata.Type)
	for _, cfg := range []any{struct{}{}, (*config.Config)(nil)} {
		_, _, err := manager.Start(context.Background(), settings, cfg)
		require.Error(t, err)
	}
	cfg := config.NewConfigWithDefaults()
	cfg.Collectors = map[string]bool{"does_not_exist": true}
	_, _, err := manager.Start(context.Background(), settings, &cfg)
	require.ErrorContains(t, err, "unknown collector")
}

type contextCollector struct {
	contexts chan context.Context
	block    bool
}

func (c *contextCollector) Update(ctx context.Context, _ chan<- prometheus.Metric) error {
	c.contexts <- ctx
	if c.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func TestLifecycleShutdownCancelsOnlyItsRuntime(t *testing.T) {
	t.Parallel()
	var collectors []*contextCollector
	manager := &lifecycleManager{newRuntime: func(cfg config.Config, logger *slog.Logger) (exporterRuntime, error) {
		runtime, err := nodecollector.NewRuntime(cfg, logger)
		if err != nil {
			return nil, err
		}
		// Keep the real upstream runtime/context plumbing and replace only the
		// data source, so cancellation can be observed without a system D-Bus.
		c := &contextCollector{contexts: make(chan context.Context, 1), block: len(collectors) == 0}
		collectors = append(collectors, c)
		runtime.Collectors()[0].(*nodecollector.NodeCollector).Collectors = map[string]nodecollector.Collector{"context": c}
		return runtime, nil
	}}
	cfg := config.NewConfigWithDefaults()
	cfg.CollectorDisableDefaults = true
	settings := receivertest.NewNopSettings(metadata.Type)
	startup, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()
	first, stopFirst, err := manager.Start(startup, settings, &cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopFirst(context.Background())) })
	second, stopSecond, err := manager.Start(startup, settings, &cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopSecond(context.Background())) })
	cancelStartup()

	gathered := make(chan error, 1)
	go func() { _, err := first.Gather(); gathered <- err }()
	var firstContext context.Context
	select {
	case firstContext = <-collectors[0].contexts:
	case <-time.After(5 * time.Second):
		t.Fatal("first runtime did not start collection")
	}
	require.NoError(t, firstContext.Err(), "startup cancellation must not cancel collection")
	_, err = second.Gather()
	require.NoError(t, err)
	secondContext := <-collectors[1].contexts
	require.NoError(t, secondContext.Err())

	require.NoError(t, stopFirst(context.Background()))
	require.ErrorIs(t, firstContext.Err(), context.Canceled)
	require.NoError(t, secondContext.Err(), "another runtime must remain active")
	select {
	case err := <-gathered:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("runtime shutdown did not unblock collection")
	}
	require.NoError(t, stopFirst(context.Background()), "shutdown must be repeatable")
	require.NoError(t, stopSecond(context.Background()))
	require.ErrorIs(t, secondContext.Err(), context.Canceled)
}

type registrationFailureRuntime struct {
	shutdownCalled bool
	shutdownErr    error
}

func (*registrationFailureRuntime) Collectors() []prometheus.Collector {
	// Registration rejects duplicate descriptor label names.
	return []prometheus.Collector{prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "invalid"}, []string{"duplicate", "duplicate"})}
}

func (r *registrationFailureRuntime) Shutdown(context.Context) error {
	r.shutdownCalled = true
	return r.shutdownErr
}

func TestLifecycleRegistrationFailureCleansUpRuntime(t *testing.T) {
	t.Parallel()
	for _, shutdownErr := range []error{nil, errors.New("shutdown failed")} {
		runtime := &registrationFailureRuntime{shutdownErr: shutdownErr}
		manager := &lifecycleManager{newRuntime: func(config.Config, *slog.Logger) (exporterRuntime, error) {
			return runtime, nil
		}}
		cfg := config.NewConfigWithDefaults()
		registry, shutdown, err := manager.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), &cfg)
		require.ErrorContains(t, err, "register collector")
		require.Nil(t, registry)
		require.Nil(t, shutdown)
		require.True(t, runtime.shutdownCalled)
		if shutdownErr != nil {
			require.ErrorIs(t, err, shutdownErr)
		}
	}
}
