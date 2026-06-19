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

package mysqld

import (
	"context"
	"reflect"
	"testing"

	"github.com/prometheus/mysqld_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/mysqld/internal/metadata"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}
}

func createMetrics(t *testing.T, exporterConfig map[string]interface{}) error {
	t.Helper()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ExporterConfig = exporterConfig

	_, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		new(consumertest.MetricsSink),
	)
	return err
}

// TestFactory_DefaultConfigDerivesExporterDefaults confirms the bridge derives
// rendered defaults straight from config.Config.
func TestFactory_DefaultConfigDerivesExporterDefaults(t *testing.T) {
	t.Parallel()

	cfg := NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)

	want := map[string]interface{}{
		"data_source_name":                  "",
		"collectors":                        config.DefaultCollectorConfig(),
		"timeout_offset":                    config.DefaultTimeoutOffset,
		"enable_exporter_lock_wait_timeout": config.DefaultEnableExporterLockTimeout,
		"exporter_lock_wait_timeout":        config.DefaultExporterLockWaitTimeout,
		"slow_log_filter":                   config.DefaultSlowLogFilter,
		"heartbeat": config.HeartbeatConfig{
			Database: config.DefaultHeartbeatDatabase,
			Table:    config.DefaultHeartbeatTable,
			UTC:      config.DefaultHeartbeatUTC,
		},
		"info_schema_processlist": config.InfoSchemaProcesslistConfig{
			MinTime:         config.DefaultInfoSchemaProcesslistMinTime,
			ProcessesByUser: config.DefaultInfoSchemaProcesslistProcessesByUser,
			ProcessesByHost: config.DefaultInfoSchemaProcesslistProcessesByHost,
		},
		"info_schema_tables": config.InfoSchemaTablesConfig{
			Databases: config.DefaultInfoSchemaTablesDatabases,
		},
		"perf_schema_events_statements": config.PerfSchemaEventsStatementsConfig{
			Limit:           config.DefaultPerfSchemaEventsStatementsLimit,
			TimeLimit:       config.DefaultPerfSchemaEventsStatementsTimeLimit,
			DigestTextLimit: config.DefaultPerfSchemaEventsStatementsDigestTextLimit,
			ExcludeSchemas:  []string(nil),
		},
		"perf_schema_file_instances": config.PerfSchemaFileInstancesConfig{
			Filter:       config.DefaultPerfSchemaFileInstancesFilter,
			RemovePrefix: config.DefaultPerfSchemaFileInstancesRemovePrefix,
		},
		"perf_schema_memory_events": config.PerfSchemaMemoryEventsConfig{
			RemovePrefix: config.DefaultPerfSchemaMemoryEventsRemovePrefix,
		},
		"mysql_user": config.MysqlUserConfig{
			Privileges: config.DefaultMysqlUserPrivileges,
		},
	}

	if !reflect.DeepEqual(cfg.ExporterConfig, want) {
		t.Fatalf("derived exporter defaults mismatch.\n got: %#v\nwant: %#v", cfg.ExporterConfig, want)
	}
}

// TestFactory_CreateMetrics_DecodesUntaggedConfig exercises the bridge wiring:
// snake_case keys map onto config.Config's fields, including nested structs.
func TestFactory_CreateMetrics_DecodesUntaggedConfig(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"data_source_name":                  "exporter:secret@(127.0.0.1:3306)/",
		"timeout_offset":                    0.5,
		"enable_exporter_lock_wait_timeout": false,
		"exporter_lock_wait_timeout":        5,
		"slow_log_filter":                   true,
		"collectors": map[string]interface{}{
			"global_status":    true,
			"global_variables": true,
			"slave_status":     false,
		},
		"heartbeat": map[string]interface{}{
			"database": "custom_heartbeat",
			"table":    "custom_table",
			"utc":      true,
		},
		"info_schema_processlist": map[string]interface{}{
			"min_time":          3,
			"processes_by_user": false,
			"processes_by_host": false,
		},
		"info_schema_tables": map[string]interface{}{
			"databases": "app",
		},
		"perf_schema_events_statements": map[string]interface{}{
			"limit":             100,
			"time_limit":        3600,
			"digest_text_limit": 80,
			"exclude_schemas":   []string{"mysql"},
		},
		"perf_schema_file_instances": map[string]interface{}{
			"filter":        ".*ibd",
			"remove_prefix": "/data/mysql/",
		},
		"perf_schema_memory_events": map[string]interface{}{
			"remove_prefix": "memory/sql/",
		},
		"mysql_user": map[string]interface{}{
			"privileges": true,
		},
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

func TestFactory_CreateMetrics_MissingDataSourceName(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownCollector(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"data_source_name": "exporter:secret@(127.0.0.1:3306)/",
		"collectors": map[string]interface{}{
			"not_a_real_collector": true,
		},
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownField(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"data_source_name": "exporter:secret@(127.0.0.1:3306)/",
		"not_a_real_field": "value",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want unknown-field failure")
	}
}
