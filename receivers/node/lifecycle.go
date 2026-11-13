// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	nodecollector "github.com/prometheus/node_exporter/collector"
	"github.com/prometheus/node_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type exporterRuntime interface {
	Collectors() []prometheus.Collector
	Shutdown(context.Context) error
}

type lifecycleManager struct {
	newRuntime func(config.Config, *slog.Logger) (exporterRuntime, error)
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{
		newRuntime: func(cfg config.Config, logger *slog.Logger) (exporterRuntime, error) {
			return nodecollector.NewRuntime(cfg, logger)
		},
	}
}

func (m *lifecycleManager) Start(ctx context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := exporterCfg.(*config.Config)
	if !ok || cfg == nil {
		return nil, nil, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	// The startup context does not bound the receiver's lifetime. The bridge
	// retains the shutdown function for this particular runtime instead.
	runtime, err := m.newRuntime(*cfg, collectorSlogLogger(set))
	if err != nil {
		return nil, nil, fmt.Errorf("start node runtime: %w", err)
	}

	registry := prometheus.NewRegistry()
	for _, c := range runtime.Collectors() {
		if err := registry.Register(c); err != nil {
			if shutdownErr := runtime.Shutdown(ctx); shutdownErr != nil {
				return nil, nil, fmt.Errorf("register collector: %w; shutdown node runtime: %w", err, shutdownErr)
			}
			return nil, nil, fmt.Errorf("register collector: %w", err)
		}
	}

	return registry, runtime.Shutdown, nil
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
