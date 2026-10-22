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

package yace

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/metrics"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/promutil"
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

func (m *lifecycleManager) Start(_ context.Context, set receiver.Settings, exporterCfg any) (*prometheus.Registry, component.ShutdownFunc, error) {
	cfg, ok := exporterCfg.(*config.Config)
	if !ok {
		return nil, nil, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	logger := m.loggerFromSettings(set)
	scrapeConf := config.ScrapeConf{}
	jobsCfg, err := scrapeConf.Load(cfg.ScrapeConfigFile, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("load YACE scrape config: %w", err)
	}

	registry := prometheus.NewRegistry()
	scrapeMetrics := promutil.NewScrapeMetrics(registry)

	factory, err := clients.NewFactory(logger, scrapeMetrics, jobsCfg, cfg.FIPSEnabled)
	if err != nil {
		return nil, nil, fmt.Errorf("create YACE client factory: %w", err)
	}

	scraper, err := metrics.NewScraper(logger, scrapeMetrics, *cfg, jobsCfg, factory)
	if err != nil {
		return nil, nil, fmt.Errorf("create YACE scraper: %w", err)
	}

	// The Start context is only for startup work; scrapes happen later from
	// prometheus.Collector. Keep a component-lifetime context so Shutdown can
	// cancel in-flight CloudWatch calls.
	scrapeCtx, cancel := context.WithCancel(context.Background())
	if err := registry.Register(newScrapeCollector(scrapeCtx, logger, scraper)); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("register YACE scrape collector: %w", err)
	}

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
