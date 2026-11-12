// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

// A real HTTP connection makes upstream resource ownership observable through
// the lifecycle adapter, without depending on internal runtime state.
func TestLifecycleShutdownClosesOnlyItsRuntime(t *testing.T) {
	t.Parallel()
	manager := newLifecycleManager()
	settings := receivertest.NewNopSettings(metadata.Type)
	startup, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()
	registries := make([]*prometheus.Registry, 2)
	stops := make([]func(context.Context) error, 2)
	closed := make([]atomic.Int32, 2)
	for i := range registries {
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><array><data><value><struct>
<member><name>name</name><value><string>fixture</string></value></member>
<member><name>group</name><value><string>fixture</string></value></member>
<member><name>state</name><value><int>20</int></value></member>
</struct></value></data></array></value></param></params></methodResponse>`)
		}))
		srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
			if state == http.StateClosed {
				closed[i].Add(1)
			}
		}
		srv.Start()
		t.Cleanup(srv.Close)
		cfg := config.NewConfigWithDefaults()
		cfg.CollectorDisableDefaults = true
		cfg.Collectors = map[string]bool{"supervisord": true}
		cfg.Supervisord.URL = srv.URL + "/RPC2"
		registry, stop, err := manager.Start(startup, settings, &cfg)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, stop(context.Background())) })
		registries[i], stops[i] = registry, stop
	}
	cancelStartup()
	for _, registry := range registries {
		count, err := testutil.GatherAndCount(registry, "node_supervisord_up")
		require.NoError(t, err)
		require.Equal(t, 1, count, "startup cancellation must not prevent collection")
	}
	require.NoError(t, stops[0](context.Background()))
	require.Eventually(t, func() bool { return closed[0].Load() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.Zero(t, closed[1].Load(), "another runtime's connection must remain open")
	count, err := testutil.GatherAndCount(registries[1], "node_supervisord_up")
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, stops[0](context.Background()), "shutdown must be repeatable")
	require.NoError(t, stops[1](context.Background()))
	require.Eventually(t, func() bool { return closed[1].Load() == 1 }, 5*time.Second, 10*time.Millisecond)
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
