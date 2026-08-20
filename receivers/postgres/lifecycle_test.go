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

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), wrongConfig{})
	if err == nil {
		t.Fatal("Start() expected error for wrong config type, got nil")
	}
}

func TestLifecycleManager_Start_ValidatesConfig(t *testing.T) {
	t.Parallel()

	defaults := exporterConfig(config.NewConfigWithDefaults())
	cfg := &defaults
	cfg.MetricPrefix = ""

	mgr := newLifecycleManager()
	mgr.newRuntime = func(config.ValidatedConfig, *slog.Logger) (runtime, error) {
		t.Fatal("newRuntime should not be called for invalid config")
		return nil, nil
	}

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
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

	registry, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if registry == nil {
		t.Fatal("Start() returned nil registry")
	}

	if _, err := registry.Gather(); err != nil {
		t.Fatalf("registry.Gather() error = %v", err)
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

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err == nil {
		t.Fatal("Start() expected register error, got nil")
	}
	if !fake.closed {
		t.Fatal("runtime was not closed after register error")
	}
}

func TestLifecycleManager_ShutdownClosesRuntime(t *testing.T) {
	t.Parallel()

	runtime := &fakeRuntime{}
	mgr := newLifecycleManager()
	mgr.runtime = runtime

	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if !runtime.closed {
		t.Fatal("Shutdown() did not close runtime")
	}
}

func TestLifecycleManager_ShutdownWithoutRuntime(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()
	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
