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
	"reflect"
	"testing"

	"github.com/prometheus/mysqld_exporter/config"
)

// TestConfigStructShape pins the shape of the upstream mysqld config.Config.
// The bridge derives OTel keys and component defaults from this struct by
// reflection, so upstream changes should force a receiver review.
func TestConfigStructShape(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"DataSourceName":                "string",
		"Collectors":                    "map[string]bool",
		"TimeoutOffset":                 "float64",
		"EnableExporterLockWaitTimeout": "bool",
		"ExporterLockWaitTimeout":       "int",
		"SlowLogFilter":                 "bool",
		"Heartbeat":                     "config.HeartbeatConfig",
		"InfoSchemaProcesslist":         "config.InfoSchemaProcesslistConfig",
		"InfoSchemaTables":              "config.InfoSchemaTablesConfig",
		"PerfSchemaEventsStatements":    "config.PerfSchemaEventsStatementsConfig",
		"PerfSchemaFileInstances":       "config.PerfSchemaFileInstancesConfig",
		"PerfSchemaMemoryEvents":        "config.PerfSchemaMemoryEventsConfig",
		"MysqlUser":                     "config.MysqlUserConfig",
	}

	got := map[string]string{}
	rt := reflect.TypeOf(config.Config{})
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		got[f.Name] = f.Type.String()
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config.Config shape changed; review the OTel key mapping, metadata.yaml, and docs, then update this golden.\n got: %#v\nwant: %#v", got, want)
	}
}
