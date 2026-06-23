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
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/prometheus-community/postgres_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

func postgresOTTLStatements(exporterCfg any) (prombridge.OTTLStatements, error) {
	cfg, ok := exporterCfg.(*config.Config)
	if !ok {
		return prombridge.OTTLStatements{}, fmt.Errorf("expected *config.Config, got %T", exporterCfg)
	}

	server, err := serverFromConfig(cfg)
	if err != nil {
		return prombridge.OTTLStatements{}, err
	}

	return prombridge.OTTLStatements{
		MetricStatements: postgresMetricStatements(),
		DataPointStatements: append(
			postgresDataPointStatements(),
			server.ottlStatements()...,
		),
	}, nil
}

func postgresMetricStatements() []string {
	return []string{
		`set(metric.unit, "By") where IsMatch(metric.name, "^pg_.*_bytes(_total)?$")`,
		`set(metric.unit, "s") where IsMatch(metric.name, "^pg_.*_seconds(_total)?$")`,
		`set(metric.unit, "s") where metric.name == "pg_stat_activity_max_tx_duration"`,
		`set(metric.unit, "ms") where metric.name == "pg_stat_database_blk_read_time_total"`,
		`set(metric.unit, "ms") where metric.name == "pg_stat_database_blk_write_time_total"`,
		`replace_pattern(metric.name, "^pg_", "db.server.postgresql.") where IsMatch(metric.name, "^pg_")`,
		`replace_pattern(metric.name, "_total$", "") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
	}
}

func postgresDataPointStatements() []string {
	return []string{
		`set(datapoint.attributes["db.system.name"], "postgresql") where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
		`set(datapoint.attributes["db.namespace"], datapoint.attributes["datname"]) where datapoint.attributes["datname"] != nil`,
		`set(datapoint.attributes["db.collection.name"], Concat([datapoint.attributes["schemaname"], datapoint.attributes["relname"]], ".")) where datapoint.attributes["schemaname"] != nil and datapoint.attributes["relname"] != nil`,
		`set(datapoint.attributes["db.client.connection.state"], "idle") where datapoint.attributes["state"] == "idle"`,
		`set(datapoint.attributes["db.client.connection.state"], "used") where datapoint.attributes["state"] == "active"`,
	}
}

type serverIdentity struct {
	address  string
	port     int
	database string
}

func serverFromConfig(cfg *config.Config) (serverIdentity, error) {
	if len(cfg.DataSourceNames) == 0 {
		return serverIdentity{}, nil
	}

	dsn := cfg.DataSourceNames[0]
	if !strings.Contains(dsn, "://") {
		return serverIdentity{}, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return serverIdentity{}, fmt.Errorf("parse postgres data source name: %w", err)
	}

	port := 5432
	if u.Port() != "" {
		parsedPort, err := strconv.Atoi(u.Port())
		if err != nil {
			return serverIdentity{}, fmt.Errorf("parse postgres data source port: %w", err)
		}
		port = parsedPort
	}

	return serverIdentity{
		address:  u.Hostname(),
		port:     port,
		database: strings.TrimPrefix(u.EscapedPath(), "/"),
	}, nil
}

func (s serverIdentity) ottlStatements() []string {
	if s.address == "" {
		return nil
	}

	statements := []string{
		fmt.Sprintf(
			`set(datapoint.attributes["server.address"], %s) where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
			strconv.Quote(s.address),
		),
		fmt.Sprintf(
			`set(datapoint.attributes["server.port"], %d) where IsMatch(metric.name, "^db\\.server\\.postgresql\\.")`,
			s.port,
		),
	}
	if s.database != "" {
		poolName := fmt.Sprintf("%s:%d/%s", s.address, s.port, s.database)
		statements = append(statements, fmt.Sprintf(
			`set(datapoint.attributes["db.client.connection.pool.name"], %s) where IsMatch(metric.name, "^db\\.server\\.postgresql\\.stat\\.activity\\.count$")`,
			strconv.Quote(poolName),
		))
	}
	return statements
}
