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

package stackdriver

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus-community/stackdriver_exporter/collectors"
	"github.com/prometheus-community/stackdriver_exporter/config"
	"github.com/prometheus-community/stackdriver_exporter/delta"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/exp/zapslog"
)

var _ prombridge.ExporterLifecycleManager = (*lifecycleManager)(nil)

type lifecycleManager struct {
	loggerFromSettings func(set receiver.Settings) *slog.Logger
}

func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{loggerFromSettings: collectorSlogLogger}
}

func (m *lifecycleManager) Start(ctx context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := exporterCfg.(*config.Config)
	if !ok {
		return nil, nil, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	// The Google client's token source can retain this context after startup.
	// Honor startup cancellation while initializing, then let receiver shutdown
	// own cancellation independently of the Collector's Start context.
	runtimeCtx, cancel := context.WithCancel(context.Background())
	stopStartupCancellation := context.AfterFunc(ctx, cancel)
	defer stopStartupCancellation()
	started := false
	defer func() {
		if !started {
			cancel()
		}
	}()
	logger := m.loggerFromSettings(set)
	runtime, err := collectors.NewRuntime(runtimeCtx, logger, cfg, delta.NewInMemoryCounterStore, delta.NewInMemoryHistogramStore)
	if err != nil {
		return nil, nil, fmt.Errorf("start stackdriver runtime: %w", err)
	}

	collectorList, err := runtime.Collectors()
	if err != nil {
		return nil, nil, fmt.Errorf("build collectors: %w", err)
	}

	registry := prometheus.NewRegistry()
	for _, c := range collectorList {
		if err := registry.Register(c); err != nil {
			return nil, nil, fmt.Errorf("register collector: %w", err)
		}
	}
	stopStartupCancellation()
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("start stackdriver runtime: %w", err)
	}
	started = true
	return registry, func(context.Context) error {
		cancel()
		return nil
	}, nil
}

func collectorSlogLogger(set receiver.Settings) *slog.Logger {
	if set.Logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return slog.New(zapslog.NewHandler(set.Logger.Core()))
}
