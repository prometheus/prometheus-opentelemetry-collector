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
	"reflect"
	"testing"

	"github.com/prometheus-community/postgres_exporter/config"
)

func TestConfigUnmarshaler_GetConfigStruct(t *testing.T) {
	t.Parallel()

	want := exporterConfig(config.NewConfigWithDefaults())

	got, ok := configUnmarshaler{}.GetConfigStruct().(*exporterConfig)
	if !ok {
		t.Fatalf("GetConfigStruct() returned unexpected type %T", got)
	}

	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("GetConfigStruct() = %#v, want %#v", got, want)
	}
}

func TestExporterConfig_Validate(t *testing.T) {
	t.Parallel()

	cfg := exporterConfig(config.NewConfigWithDefaults())
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	cfg.MetricPrefix = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error for empty metric prefix")
	}
}

func TestExporterConfig_Validated(t *testing.T) {
	t.Parallel()

	cfg := exporterConfig(config.NewConfigWithDefaults())
	cfg.MetricPrefix = "custompg"

	validated, err := cfg.validated()
	if err != nil {
		t.Fatalf("validated() error = %v", err)
	}
	if !validated.Valid() {
		t.Fatal("validated() returned a config that is not marked valid")
	}
	if got := validated.Config().MetricPrefix; got != "custompg" {
		t.Fatalf("validated config MetricPrefix = %q, want %q", got, "custompg")
	}

	cfg.MetricPrefix = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("validated() error = nil, want error for empty metric prefix")
	}
}
