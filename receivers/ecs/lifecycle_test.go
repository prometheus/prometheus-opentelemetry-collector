// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/ecs/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type fakeRuntime struct {
	collectors  []prometheus.Collector
	shutdown    bool
	shutdownErr error
}

func (r *fakeRuntime) Collectors() []prometheus.Collector { return r.collectors }

func (r *fakeRuntime) Shutdown(context.Context) error {
	r.shutdown = true
	return r.shutdownErr
}

func TestLifecycleManagerRejectsWrongConfig(t *testing.T) {
	t.Parallel()

	manager := newLifecycleManager()
	if _, err := manager.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), struct{}{}); err == nil {
		t.Fatal("Start() accepted the wrong config type")
	}
}

func TestLifecycleManagerReportsMissingMetadataEndpoint(t *testing.T) {
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", "")

	manager := newLifecycleManager()
	if _, err := manager.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), &exporterConfig{}); err == nil {
		t.Fatal("Start() succeeded without ECS_CONTAINER_METADATA_URI_V4")
	}
}

func TestLifecycleManagerShutsDownRuntime(t *testing.T) {
	t.Parallel()

	fake := &fakeRuntime{}
	manager := newLifecycleManager()
	manager.runtime = fake
	if err := manager.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if !fake.shutdown {
		t.Fatal("Shutdown() did not shut down the runtime")
	}
}

func TestLifecycleManagerEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /task", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"Cluster": "test-cluster",
			"TaskARN": "task-arn",
			"Family": "family",
			"Revision": "1",
			"DesiredStatus": "RUNNING",
			"KnownStatus": "RUNNING",
			"AvailabilityZone": "us-east-1a",
			"LaunchType": "FARGATE",
			"Containers": []
		}`)
	})
	mux.HandleFunc("GET /task/stats", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)

	manager := newLifecycleManager()
	registry, err := manager.Start(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		&exporterConfig{},
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	expected := `# HELP ecs_task_metadata_info ECS task metadata, sourced from the task metadata endpoint version 4.
# TYPE ecs_task_metadata_info gauge
ecs_task_metadata_info{availability_zone="us-east-1a",cluster="test-cluster",desired_status="RUNNING",family="family",known_status="RUNNING",launch_type="FARGATE",revision="1",task_arn="task-arn"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "ecs_task_metadata_info"); err != nil {
		t.Fatalf("unexpected ECS metrics: %v", err)
	}
}

func TestLifecycleManagerStartsRuntime(t *testing.T) {
	t.Parallel()

	fake := &fakeRuntime{collectors: []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "ecs_receiver_test_metric", Help: "test metric"}),
	}}
	manager := newLifecycleManager()
	manager.newClientFromEnvironment = func() (*ecsmetadata.Client, error) {
		return ecsmetadata.NewClient("http://example.invalid"), nil
	}
	manager.newRuntime = func(_ *ecsmetadata.Client, logger *slog.Logger) runtime {
		if logger == nil {
			t.Fatal("newRuntime received a nil logger")
		}
		return fake
	}

	registry, err := manager.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), &exporterConfig{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := registry.Gather(); err != nil {
		t.Fatalf("registry.Gather() error = %v", err)
	}
}
