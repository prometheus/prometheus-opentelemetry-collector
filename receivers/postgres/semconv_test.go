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
	"strings"
	"testing"

	"github.com/prometheus-community/postgres_exporter/config"
)

func TestPostgresOTTLStatements_DerivesServerAttributesFromDSN(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfigWithDefaults()
	cfg.DataSourceNames = []string{"postgresql://user:pass@db.example.test:15432/demo?sslmode=disable"}

	statements, err := postgresOTTLStatements(&cfg)
	if err != nil {
		t.Fatalf("postgresOTTLStatements() error = %v", err)
	}

	if !containsStatement(statements.DataPointStatements, `set(datapoint.attributes["server.address"], "db.example.test")`) {
		t.Fatalf("server.address statement not found in %#v", statements.DataPointStatements)
	}
	if !containsStatement(statements.DataPointStatements, `set(datapoint.attributes["server.port"], 15432)`) {
		t.Fatalf("server.port statement not found in %#v", statements.DataPointStatements)
	}
	if !containsStatement(statements.DataPointStatements, `set(datapoint.attributes["db.client.connection.pool.name"], "db.example.test:15432/demo")`) {
		t.Fatalf("pool-name statement not found in %#v", statements.DataPointStatements)
	}
}

func TestPostgresOTTLStatements_DoesNotUsePrivateTransformProcessorFunctions(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfigWithDefaults()
	cfg.DataSourceNames = []string{"postgresql://user:pass@localhost:5432/demo?sslmode=disable"}

	statements, err := postgresOTTLStatements(&cfg)
	if err != nil {
		t.Fatalf("postgresOTTLStatements() error = %v", err)
	}

	allStatements := strings.Join(append(statements.MetricStatements, statements.DataPointStatements...), "\n")
	if strings.Contains(allStatements, "scale_metric") {
		t.Fatal("embedded receiver OTTL must not use transformprocessor-internal scale_metric")
	}
}

func containsStatement(statements []string, want string) bool {
	for _, statement := range statements {
		if strings.Contains(statement, want) {
			return true
		}
	}
	return false
}
