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

package elasticsearch

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus-community/elasticsearch_exporter/collector"
	"github.com/prometheus-community/elasticsearch_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type runtime interface {
	Start(context.Context) error
	Collectors() ([]prometheus.Collector, error)
	Close() error
}

type lifecycleManager struct {
	loggerFromSettings func(set receiver.Settings) *slog.Logger

	// newRuntime is injectable so lifecycle tests can exercise startup,
	// collector registration, and shutdown without constructing a real
	// Elasticsearch exporter runtime.
	newRuntime func(ctx context.Context, logger *slog.Logger, cfg config.Config) (runtime, error)
	runtime    runtime
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{
		loggerFromSettings: collectorSlogLogger,
		newRuntime: func(ctx context.Context, logger *slog.Logger, cfg config.Config) (runtime, error) {
			return collector.NewRuntime(ctx, logger, cfg)
		},
	}
}

func (m *lifecycleManager) Start(ctx context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, error) {
	cfg, ok := exporterCfg.(*config.Config)
	if !ok {
		return nil, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	logger := m.loggerFromSettings(set)
	runtime, err := m.newRuntime(ctx, logger, *cfg)
	if err != nil {
		return nil, fmt.Errorf("start elasticsearch runtime: %w", err)
	}
	if err := runtime.Start(ctx); err != nil {
		_ = runtime.Close()
		return nil, fmt.Errorf("start elasticsearch background tasks: %w", err)
	}

	collectorList, err := runtime.Collectors()
	if err != nil {
		_ = runtime.Close()
		return nil, fmt.Errorf("build collectors: %w", err)
	}

	registry := prometheus.NewRegistry()
	for _, c := range collectorList {
		if err := registry.Register(c); err != nil {
			_ = runtime.Close()
			return nil, fmt.Errorf("register collector: %w", err)
		}
	}
	m.runtime = runtime
	return registry, nil
}

func (m *lifecycleManager) Shutdown(context.Context) error {
	if m.runtime == nil {
		return nil
	}
	err := m.runtime.Close()
	m.runtime = nil
	return err
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
