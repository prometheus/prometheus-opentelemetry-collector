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
	"slices"
	"testing"

	"github.com/prometheus-community/postgres_exporter/config"
)

func TestPostgreSQLSemconvOTTLStatements_DerivesServerAttributesFromURLDSN(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfigWithDefaults()
	cfg.DataSourceNames = []string{"postgresql://postgres:postgres@db.example.com:15432/demo?sslmode=disable"}

	statements, err := postgresqlSemconvOTTLStatements(&cfg)
	if err != nil {
		t.Fatalf("postgresqlSemconvOTTLStatements() error = %v", err)
	}

	wantDatapointStatements := []string{
		`set(datapoint.attributes["service.instance.id"], "db.example.com:15432")`,
		`set(datapoint.attributes["server.address"], "db.example.com")`,
		`set(datapoint.attributes["server.port"], 15432)`,
		`set(datapoint.attributes["db.system.name"], "postgresql")`,
	}
	for _, want := range wantDatapointStatements {
		if !slices.Contains(statements.DataPointStatements, want) {
			t.Fatalf("DataPointStatements missing %q", want)
		}
	}
}

func TestPostgreSQLSemconvOTTLStatements_OmitsPortAttributesWhenDSNHasNoPort(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfigWithDefaults()
	cfg.DataSourceNames = []string{"postgresql://postgres:postgres@db.example.com/demo?sslmode=disable"}

	statements, err := postgresqlSemconvOTTLStatements(&cfg)
	if err != nil {
		t.Fatalf("postgresqlSemconvOTTLStatements() error = %v", err)
	}

	if !slices.Contains(statements.DataPointStatements, `set(datapoint.attributes["server.address"], "db.example.com")`) {
		t.Fatal("DataPointStatements missing server.address")
	}
	for _, unwanted := range []string{
		`set(datapoint.attributes["service.instance.id"], "db.example.com:5432")`,
		`set(datapoint.attributes["server.port"], 5432)`,
	} {
		if slices.Contains(statements.DataPointStatements, unwanted) {
			t.Fatalf("DataPointStatements unexpectedly contained %q", unwanted)
		}
	}
}

func TestPostgresServerFromDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want postgresServer
	}{
		{
			name: "url with explicit port",
			dsn:  "postgresql://postgres:postgres@localhost:15432/demo?sslmode=disable",
			want: postgresServer{address: "localhost", port: 15432},
		},
		{
			name: "url without port leaves port unset",
			dsn:  "postgresql://postgres:postgres@localhost/demo?sslmode=disable",
			want: postgresServer{address: "localhost"},
		},
		{
			name: "keyword dsn",
			dsn:  "host=db.example.com port=25432 user=postgres dbname=demo sslmode=disable",
			want: postgresServer{address: "db.example.com", port: 25432},
		},
		{
			name: "keyword dsn without port leaves port unset",
			dsn:  "host=db.example.com user=postgres dbname=demo sslmode=disable",
			want: postgresServer{address: "db.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := postgresServerFromDSN(tt.dsn)
			if err != nil {
				t.Fatalf("postgresServerFromDSN() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("postgresServerFromDSN() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPostgreSQLResourceAttributeKeys(t *testing.T) {
	t.Parallel()

	want := []string{
		"service.instance.id",
		"server.address",
		"server.port",
		"postgresql.database.name",
		"postgresql.schema.name",
		"postgresql.table.name",
		"postgresql.index.name",
	}
	if !reflect.DeepEqual(postgresqlResourceAttributeKeys(), want) {
		t.Fatalf("postgresqlResourceAttributeKeys() = %#v, want %#v", postgresqlResourceAttributeKeys(), want)
	}
}
