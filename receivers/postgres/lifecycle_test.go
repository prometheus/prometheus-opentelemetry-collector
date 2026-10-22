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
	"log/slog"
	"testing"

	"github.com/prometheus-community/postgres_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type wrongConfig struct{}

func (wrongConfig) Validate() error { return nil }

type fakeRuntime struct {
	collectors []prometheus.Collector
	closeErr   error
	closed     bool
}

func (r *fakeRuntime) Collectors() []prometheus.Collector { return r.collectors }

func (r *fakeRuntime) Close() error {
	r.closed = true
	return r.closeErr
}

func TestLifecycleManager_Start_RejectsWrongConfigType(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()

	_, _, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), wrongConfig{})
	if err == nil {
		t.Fatal("Start() expected error for wrong config type, got nil")
	}
}

func TestLifecycleManager_Start_ValidatesConfig(t *testing.T) {
	t.Parallel()

	defaults := exporterConfig(config.NewConfigWithDefaults())
	cfg := &defaults
	// postgres_exporter requires a non-empty metric prefix:
	// https://github.com/prometheus-community/postgres_exporter/blob/5a2bd34f79f6/config/config.go#L130-L135
	cfg.MetricPrefix = ""

	mgr := newLifecycleManager()
	mgr.newRuntime = func(config.ValidatedConfig, *slog.Logger) (runtime, error) {
		t.Fatal("newRuntime should not be called for invalid config")
		return nil, nil
	}

	_, _, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err == nil {
		t.Fatal("Start() expected validation error, got nil")
	}
}

func TestLifecycleManager_Start_RegisterCollectors(t *testing.T) {
	t.Parallel()

	defaults := exporterConfig(config.NewConfigWithDefaults())
	cfg := &defaults
	cfg.MetricPrefix = "custompg"
	fake := &fakeRuntime{collectors: []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "postgres_receiver_test_metric", Help: "test metric"}),
	}}

	mgr := newLifecycleManager()
	mgr.newRuntime = func(got config.ValidatedConfig, logger *slog.Logger) (runtime, error) {
		if !got.Valid() {
			t.Fatal("newRuntime received a config that is not marked valid")
		}
		if prefix := got.Config().MetricPrefix; prefix != "custompg" {
			t.Fatalf("newRuntime config MetricPrefix = %q, want %q", prefix, "custompg")
		}
		if logger == nil {
			t.Fatal("newRuntime logger is nil")
		}
		return fake, nil
	}

	registry, shutdown, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if registry == nil {
		t.Fatal("Start() returned nil registry")
	}

	if _, err := registry.Gather(); err != nil {
		t.Fatalf("registry.Gather() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestLifecycleManager_Start_ClosesRuntimeOnRegisterError(t *testing.T) {
	t.Parallel()

	defaults := exporterConfig(config.NewConfigWithDefaults())
	cfg := &defaults
	fake := &fakeRuntime{collectors: []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "duplicate_metric", Help: "test metric"}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "duplicate_metric", Help: "test metric"}, func() float64 { return 1 }),
	}}

	mgr := newLifecycleManager()
	mgr.newRuntime = func(config.ValidatedConfig, *slog.Logger) (runtime, error) {
		return fake, nil
	}

	_, _, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err == nil {
		t.Fatal("Start() expected register error, got nil")
	}
	if !fake.closed {
		t.Fatal("runtime was not closed after register error")
	}
}

func TestLifecycleManager_StartReturnsIndependentRuntimeShutdowns(t *testing.T) {
	t.Parallel()

	defaults := exporterConfig(config.NewConfigWithDefaults())
	cfg := &defaults
	first := &fakeRuntime{}
	second := &fakeRuntime{}
	runtimes := []runtime{first, second}
	mgr := newLifecycleManager()
	mgr.newRuntime = func(config.ValidatedConfig, *slog.Logger) (runtime, error) {
		runtime := runtimes[0]
		runtimes = runtimes[1:]
		return runtime, nil
	}
	_, shutdownFirst, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	_, shutdownSecond, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err != nil {
		t.Fatalf("second Start() error = %v", err)
	}

	if err := shutdownFirst(context.Background()); err != nil {
		t.Fatalf("first shutdown() error = %v", err)
	}
	if !first.closed {
		t.Fatal("first shutdown function did not close the first runtime")
	}
	if second.closed {
		t.Fatal("first shutdown function closed the second runtime")
	}

	if err := shutdownSecond(context.Background()); err != nil {
		t.Fatalf("second shutdown() error = %v", err)
	}
	if !second.closed {
		t.Fatal("second shutdown function did not close the second runtime")
	}
}
