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

package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus-community/postgres_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type capturingLifecycleManager struct {
	cfg any
}

func (m *capturingLifecycleManager) Start(_ context.Context, _ receiver.Settings, cfg any) (*prometheus.Registry, error) {
	m.cfg = cfg
	return prometheus.NewRegistry(), nil
}

func (*capturingLifecycleManager) Shutdown(context.Context) error { return nil }

func TestNewFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}
}

func TestFactory_CreateDefaultConfig_DerivesExporterDefaults(t *testing.T) {
	t.Parallel()

	defaults := config.NewConfigWithDefaults()
	want := map[string]interface{}{
		"data_source_names":       defaults.DataSourceNames,
		"metric_prefix":           defaults.MetricPrefix,
		"collection_timeout":      defaults.CollectionTimeout.String(),
		"disable_default_metrics": defaults.DisableDefaultMetrics,
		"auto_discover_databases": defaults.AutoDiscoverDatabases,
		"user_queries_path":       defaults.UserQueriesPath,
		"constant_labels":         defaults.ConstantLabels,
		"exclude_databases":       defaults.ExcludeDatabases,
		"include_databases":       defaults.IncludeDatabases,
		"collectors":              defaults.Collectors,
		"pg_stat_statements":      defaults.PGStatStatements,
	}

	cfg := NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)
	if !reflect.DeepEqual(cfg.ExporterConfig, want) {
		t.Fatalf("derived exporter defaults mismatch.\n got: %#v\nwant: %#v", cfg.ExporterConfig, want)
	}
}

func TestFactory_CreateMetrics(t *testing.T) {
	t.Parallel()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)

	cfg.ExporterConfig = map[string]interface{}{
		"data_source_names": []string{"postgresql://user:pass@localhost:5432/postgres?sslmode=disable"},
	}

	settings := receivertest.NewNopSettings(receiverType)
	consumer := new(consumertest.MetricsSink)

	recv, err := factory.CreateMetrics(context.Background(), settings, cfg, consumer)
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
	if recv == nil {
		t.Fatal("CreateMetrics() returned nil receiver")
	}
}

func TestFactory_CreateMetrics_DecodesUntaggedConfig(t *testing.T) {
	t.Parallel()

	lifecycleManager := &capturingLifecycleManager{}
	factory := newFactoryWithLifecycleManager(lifecycleManager)
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ExporterConfig = map[string]interface{}{
		"data_source_names":       []string{"postgresql://user:pass@localhost:5432/postgres?sslmode=disable"},
		"metric_prefix":           "custompg",
		"collection_timeout":      "45s",
		"disable_default_metrics": true,
		"auto_discover_databases": true,
		"user_queries_path":       "/tmp/queries.yaml",
		"constant_labels":         "cluster=test",
		"exclude_databases":       []string{"template0"},
		"include_databases":       []string{"postgres"},
		"collectors":              map[string]bool{config.CollectorDatabase: true, config.CollectorLocks: false},
		"pg_stat_statements": map[string]interface{}{
			"include_query":     true,
			"query_length":      uint(256),
			"limit":             uint(25),
			"exclude_databases": []string{"template1"},
			"exclude_users":     []string{"replication"},
		},
	}

	recv, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(receiverType),
		cfg,
		new(consumertest.MetricsSink),
	)
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
	if err := recv.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := recv.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	decoded, ok := lifecycleManager.cfg.(*exporterConfig)
	if !ok {
		t.Fatalf("lifecycle received unexpected config type %T", lifecycleManager.cfg)
	}
	if !reflect.DeepEqual(decoded.DataSourceNames, []string{"postgresql://user:pass@localhost:5432/postgres?sslmode=disable"}) {
		t.Errorf("DataSourceNames = %v", decoded.DataSourceNames)
	}
	if decoded.MetricPrefix != "custompg" {
		t.Errorf("MetricPrefix = %q", decoded.MetricPrefix)
	}
	if decoded.CollectionTimeout != 45*time.Second {
		t.Errorf("CollectionTimeout = %v, want 45s", decoded.CollectionTimeout)
	}
	if !decoded.DisableDefaultMetrics {
		t.Errorf("DisableDefaultMetrics = false, want true")
	}
	if !decoded.AutoDiscoverDatabases {
		t.Errorf("AutoDiscoverDatabases = false, want true")
	}
	if decoded.UserQueriesPath != "/tmp/queries.yaml" {
		t.Errorf("UserQueriesPath = %q", decoded.UserQueriesPath)
	}
	if decoded.ConstantLabels != "cluster=test" {
		t.Errorf("ConstantLabels = %q", decoded.ConstantLabels)
	}
	if !reflect.DeepEqual(decoded.ExcludeDatabases, []string{"template0"}) {
		t.Errorf("ExcludeDatabases = %v", decoded.ExcludeDatabases)
	}
	if !reflect.DeepEqual(decoded.IncludeDatabases, []string{"postgres"}) {
		t.Errorf("IncludeDatabases = %v", decoded.IncludeDatabases)
	}
	if !decoded.Collectors[config.CollectorDatabase] {
		t.Errorf("Collectors[%q] = false, want true", config.CollectorDatabase)
	}
	if decoded.Collectors[config.CollectorLocks] {
		t.Errorf("Collectors[%q] = true, want false", config.CollectorLocks)
	}
	if decoded.Collectors[config.CollectorReplication] != config.DefaultCollectorConfig()[config.CollectorReplication] {
		t.Errorf("Collectors[%q] = %v, want default %v", config.CollectorReplication, decoded.Collectors[config.CollectorReplication], config.DefaultCollectorConfig()[config.CollectorReplication])
	}
	if !decoded.PGStatStatements.IncludeQuery {
		t.Errorf("PGStatStatements.IncludeQuery = false, want true")
	}
	if decoded.PGStatStatements.QueryLength != 256 {
		t.Errorf("PGStatStatements.QueryLength = %d, want 256", decoded.PGStatStatements.QueryLength)
	}
	if decoded.PGStatStatements.Limit != 25 {
		t.Errorf("PGStatStatements.Limit = %d, want 25", decoded.PGStatStatements.Limit)
	}
	if !reflect.DeepEqual(decoded.PGStatStatements.ExcludeDatabases, []string{"template1"}) {
		t.Errorf("PGStatStatements.ExcludeDatabases = %v", decoded.PGStatStatements.ExcludeDatabases)
	}
	if !reflect.DeepEqual(decoded.PGStatStatements.ExcludeUsers, []string{"replication"}) {
		t.Errorf("PGStatStatements.ExcludeUsers = %v", decoded.PGStatStatements.ExcludeUsers)
	}
}

func TestFactory_CreateMetrics_InvalidDuration(t *testing.T) {
	t.Parallel()

	err := createMetricsError(map[string]interface{}{
		"collection_timeout": "not-a-duration",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want decode failure")
	}
}

func TestFactory_CreateMetrics_UnknownField(t *testing.T) {
	t.Parallel()

	err := createMetricsError(map[string]interface{}{
		"not_a_real_field": "value",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want unknown-field failure")
	}
}

func TestFactory_CreateMetrics_ValidationError(t *testing.T) {
	t.Parallel()

	err := createMetricsError(map[string]interface{}{
		"metric_prefix": "",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func createMetricsError(exporterConfig map[string]interface{}) error {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ExporterConfig = exporterConfig

	_, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(receiverType),
		cfg,
		new(consumertest.MetricsSink),
	)
	return err
}
