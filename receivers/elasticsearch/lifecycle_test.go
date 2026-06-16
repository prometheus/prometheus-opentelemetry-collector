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
	"errors"
	"log/slog"
	"testing"

	"github.com/prometheus-community/elasticsearch_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/elasticsearch/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type wrongConfig struct{}

func (wrongConfig) Validate() error { return nil }

type fakeRuntime struct {
	startErr error
	closeErr error
	started  bool
	closed   bool
}

func (r *fakeRuntime) Start(context.Context) error {
	r.started = true
	return r.startErr
}

func (r *fakeRuntime) Collectors() ([]prometheus.Collector, error) {
	return []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "elasticsearch_receiver_test_metric",
			Help: "Test metric.",
		}),
	}, nil
}

func (r *fakeRuntime) Close() error {
	r.closed = true
	return r.closeErr
}

func TestLifecycleManager_Start_RejectsWrongConfigType(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), wrongConfig{})
	if err == nil {
		t.Fatal("Start() expected error for wrong config type, got nil")
	}
}

func TestLifecycleManager_StartRegistersCollectors(t *testing.T) {
	t.Parallel()

	fake := &fakeRuntime{}
	mgr := &lifecycleManager{
		loggerFromSettings: func(receiver.Settings) *slog.Logger { return slog.Default() },
		newRuntime: func(_ context.Context, _ *slog.Logger, cfg config.Config) (runtime, error) {
			if !cfg.Validated() {
				t.Fatal("newRuntime() got unvalidated config")
			}
			return fake, nil
		},
	}
	cfg := config.NewConfigWithDefaults()
	cfg.ElasticsearchURL = "http://localhost:9200"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	registry, err := mgr.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), &cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if registry == nil {
		t.Fatal("Start() registry = nil")
	}
	if !fake.started {
		t.Fatal("Start() did not start runtime")
	}

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("Gather() metric count = %d, want 1", len(metrics))
	}
}

func TestLifecycleManager_StartClosesRuntimeOnStartError(t *testing.T) {
	t.Parallel()

	fake := &fakeRuntime{startErr: errors.New("boom")}
	mgr := &lifecycleManager{
		loggerFromSettings: func(receiver.Settings) *slog.Logger { return slog.Default() },
		newRuntime: func(context.Context, *slog.Logger, config.Config) (runtime, error) {
			return fake, nil
		},
	}
	cfg := config.NewConfigWithDefaults()

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), &cfg)
	if err == nil {
		t.Fatal("Start() error = nil, want runtime start failure")
	}
	if !fake.closed {
		t.Fatal("Start() did not close runtime after start failure")
	}
}

func TestLifecycleManager_ShutdownClosesRuntime(t *testing.T) {
	t.Parallel()

	fake := &fakeRuntime{}
	mgr := &lifecycleManager{runtime: fake}
	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if !fake.closed {
		t.Fatal("Shutdown() did not close runtime")
	}
	if mgr.runtime != nil {
		t.Fatal("Shutdown() did not clear runtime")
	}
}

func TestLifecycleManager_ShutdownWithoutRuntime(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()
	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
