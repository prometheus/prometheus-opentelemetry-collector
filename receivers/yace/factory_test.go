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
	"reflect"
	"testing"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/yace/internal/metadata"
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
		"scrape_config_file":           config.DefaultScrapeConfigFile,
		"metrics_per_query":            config.DefaultMetricsPerQuery,
		"labels_snake_case":            config.DefaultLabelsSnakeCase,
		"tagging_api_concurrency":      config.DefaultTaggingAPIConcurrency,
		"feature_flags":                []string{},
		"fips_enabled":                 false,
		"disable_account_alias_lookup": false,
		"cloudwatch_concurrency":       config.DefaultCloudwatchConcurrency,
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
		"scrape_config_file":           "config.yml",
		"metrics_per_query":            250,
		"labels_snake_case":            true,
		"tagging_api_concurrency":      3,
		"feature_flags":                []string{"aws-sdk-v2"},
		"fips_enabled":                 true,
		"disable_account_alias_lookup": true,
		"cloudwatch_concurrency": map[string]interface{}{
			"single_limit":          7,
			"per_api_limit_enabled": false,
			"list_metrics":          7,
			"get_metric_data":       7,
			"get_metric_statistics": 7,
		},
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

func TestFactory_CreateMetrics_InvalidMetricsPerQuery(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"metrics_per_query": 0,
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownField(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"not_a_real_field": "value",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want unknown-field failure")
	}
}
