// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package blackbox

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/blackbox_exporter/prober"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

type probeCollector struct {
	ctx    context.Context
	logger *slog.Logger
	config *Config
	mu     sync.Mutex
}

func newProbeCollector(ctx context.Context, logger *slog.Logger, config *Config) *probeCollector {
	return &probeCollector{
		ctx:    ctx,
		logger: logger,
		config: config,
	}
}

func (*probeCollector) Describe(chan<- *prometheus.Desc) {
	// Probe metric descriptors vary by module, so this collector is unchecked.
}

func (c *probeCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var wg sync.WaitGroup
	for _, target := range c.config.Targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.collectTarget(target, ch)
		}()
	}
	wg.Wait()
}

func (c *probeCollector) collectTarget(target Target, ch chan<- prometheus.Metric) {
	module := c.config.Modules[target.Module]
	probe := prober.Probers[module.Prober]

	timeout := c.config.MaxTimeout - c.config.ProbeTimeoutOffset
	if module.Timeout > 0 && module.Timeout < timeout {
		timeout = module.Timeout
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	registry := prometheus.NewRegistry()
	probeSuccess := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})
	probeDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})
	registry.MustRegister(probeSuccess, probeDuration)

	start := time.Now()
	success := probe(ctx, target.Address, module, registry, c.logger.With(
		"module", target.Module,
		"target", target.Address,
		"target_name", target.Name,
	))
	probeDuration.Set(time.Since(start).Seconds())
	if success {
		probeSuccess.Set(1)
	}

	families, err := registry.Gather()
	if err != nil {
		c.logger.Error("failed to gather probe metrics",
			"err", err,
			"module", target.Module,
			"target", target.Address,
			"target_name", target.Name,
		)
	}

	labels := make(prometheus.Labels, len(target.Labels)+3)
	for name, value := range target.Labels {
		labels[name] = value
	}
	labels["module"] = target.Module
	labels["target"] = target.Address
	labels["target_name"] = target.Name

	for _, family := range families {
		for _, metric := range family.Metric {
			ch <- newMetricWithLabels(family, metric, labels)
		}
	}
}

type metricWithLabels struct {
	desc   *prometheus.Desc
	metric *dto.Metric
	labels prometheus.Labels
}

func newMetricWithLabels(family *dto.MetricFamily, metric *dto.Metric, labels prometheus.Labels) prometheus.Metric {
	variableLabels := make([]string, 0, len(metric.Label))
	for _, pair := range metric.Label {
		variableLabels = append(variableLabels, pair.GetName())
	}
	sort.Strings(variableLabels)
	return &metricWithLabels{
		desc:   prometheus.NewDesc(family.GetName(), family.GetHelp(), variableLabels, labels),
		metric: metric,
		labels: labels,
	}
}

func (m *metricWithLabels) Desc() *prometheus.Desc {
	return m.desc
}

func (m *metricWithLabels) Write(out *dto.Metric) error {
	proto.Reset(out)
	proto.Merge(out, m.metric)
	labelNames := make([]string, 0, len(m.labels))
	for name := range m.labels {
		labelNames = append(labelNames, name)
	}
	sort.Strings(labelNames)
	for _, name := range labelNames {
		name, value := name, m.labels[name]
		out.Label = append(out.Label, &dto.LabelPair{Name: &name, Value: &value})
	}
	sort.Slice(out.Label, func(i, j int) bool {
		return out.Label[i].GetName() < out.Label[j].GetName()
	})
	return nil
}
