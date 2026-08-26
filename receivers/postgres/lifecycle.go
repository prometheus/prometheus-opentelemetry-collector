// Copyright The Prometheus Authors
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

package postgres

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/prometheus-community/postgres_exporter/collector"
	"github.com/prometheus-community/postgres_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type runtime interface {
	Collectors() []prometheus.Collector
	Close() error
}

type lifecycleManager struct {
	loggerFromSettings func(set receiver.Settings) *slog.Logger
	newRuntime         func(cfg config.ValidatedConfig, logger *slog.Logger) (runtime, error)

	mu      sync.Mutex
	runtime runtime
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{
		loggerFromSettings: collectorSlogLogger,
		newRuntime: func(cfg config.ValidatedConfig, logger *slog.Logger) (runtime, error) {
			return collector.NewRuntime(cfg, logger)
		},
	}
}

func (m *lifecycleManager) Start(_ context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, error) {
	cfg, ok := exporterCfg.(*exporterConfig)
	if !ok {
		return nil, fmt.Errorf("expected *exporterConfig, got %T", exporterCfg)
	}
	validatedCfg, err := cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("validate postgres config: %w", err)
	}

	runtime, err := m.newRuntime(validatedCfg, m.loggerFromSettings(set))
	if err != nil {
		return nil, fmt.Errorf("start postgres runtime: %w", err)
	}

	registry := prometheus.NewRegistry()
	for _, c := range runtime.Collectors() {
		if err := registry.Register(c); err != nil {
			if closeErr := runtime.Close(); closeErr != nil {
				return nil, fmt.Errorf("register collector: %w; close postgres runtime: %w", err, closeErr)
			}
			return nil, fmt.Errorf("register collector: %w", err)
		}
	}

	m.mu.Lock()
	m.runtime = runtime
	m.mu.Unlock()

	return registry, nil
}

func (m *lifecycleManager) Shutdown(context.Context) error {
	m.mu.Lock()
	runtime := m.runtime
	m.runtime = nil
	m.mu.Unlock()

	if runtime == nil {
		return nil
	}
	return runtime.Close()
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
