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
	"reflect"
	"testing"

	"github.com/prometheus-community/elasticsearch_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/elasticsearch/internal/metadata"
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

func TestFactory_DefaultConfigDerivesExporterDefaults(t *testing.T) {
	t.Parallel()

	cfg := NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)

	want := map[string]interface{}{
		"elasticsearch_url":       config.DefaultElasticsearchURL,
		"timeout":                 config.DefaultTimeout.String(),
		"all_nodes":               config.DefaultAllNodes,
		"node":                    config.DefaultNode,
		"export_indices":          config.DefaultExportIndices,
		"export_indices_mappings": config.DefaultExportIndicesMappings,
		"export_index_aliases":    config.DefaultExportIndexAliases,
		"export_shards":           config.DefaultExportShards,
		"cluster_info_interval":   config.DefaultClusterInfoInterval.String(),
		"tls":                     config.TLSConfig{},
		"aws":                     config.AWSConfig{},
		"aws_enabled":             false,
		"username":                "",
		"password":                "",
		"api_key":                 "",
		"collectors":              config.DefaultCollectorConfig(),
		"tasks_actions":           config.DefaultTasksActions,
	}

	if !reflect.DeepEqual(cfg.ExporterConfig, want) {
		t.Fatalf("derived exporter defaults mismatch.\n got: %#v\nwant: %#v", cfg.ExporterConfig, want)
	}
}

func TestFactory_CreateMetrics(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url": "http://localhost:9200",
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

func TestFactory_CreateMetrics_DecodesUntaggedConfig(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url":     "https://localhost:9200",
		"timeout":               "30s",
		"all_nodes":             true,
		"cluster_info_interval": "10m",
		"export_indices":        true,
		"export_index_aliases":  false,
		"aws_enabled":           true,
		"tasks_actions":         "indices:data/read/*",
		"tls":                   map[string]interface{}{"insecure_skip_verify": true},
		"aws":                   map[string]interface{}{"region": "us-east-1", "role_arn": "arn:aws:iam::123456789012:role/es"},
		"collectors":            map[string]interface{}{config.CollectorTasks: true},
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

func TestFactory_CreateMetrics_InvalidDuration(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url": "http://localhost:9200",
		"timeout":           "not-a-duration",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want decode failure")
	}
}

func TestFactory_CreateMetrics_InvalidURL(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url": "ftp://localhost:9200",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownCollector(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url": "http://localhost:9200",
		"collectors":        map[string]interface{}{"not-real": true},
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownField(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"elasticsearch_url": "http://localhost:9200",
		"not_a_real_field":  "value",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want unknown-field failure")
	}
}
