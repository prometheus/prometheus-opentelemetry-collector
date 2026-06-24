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
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/prometheus-community/postgres_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

type postgresServer struct {
	address string
	port    int
}

func postgresqlSemconvOTTLStatements(exporterConfig any) (prombridge.OTTLStatements, error) {
	cfg, ok := exporterConfig.(*config.Config)
	if !ok {
		return prombridge.OTTLStatements{}, fmt.Errorf("unexpected postgres exporter config type %T", exporterConfig)
	}

	datapointStatements := []string{}
	if len(cfg.DataSourceNames) > 0 {
		server, err := postgresServerFromDSN(cfg.DataSourceNames[0])
		if err != nil {
			return prombridge.OTTLStatements{}, fmt.Errorf("parse postgres data source name: %w", err)
		}
		if server.address != "" {
			datapointStatements = append(datapointStatements,
				fmt.Sprintf(`set(datapoint.attributes["server.address"], %s)`, strconv.Quote(server.address)),
			)
		}
		if server.address != "" && server.port != 0 {
			datapointStatements = append(datapointStatements,
				fmt.Sprintf(`set(datapoint.attributes["service.instance.id"], %s)`, strconv.Quote(net.JoinHostPort(server.address, strconv.Itoa(server.port)))),
				fmt.Sprintf(`set(datapoint.attributes["server.port"], %d)`, server.port),
			)
		}
	}
	datapointStatements = append(datapointStatements, postgresqlDatapointStatements()...)

	return prombridge.OTTLStatements{
		MetricStatements:    postgresqlMetricStatements(),
		DataPointStatements: datapointStatements,
	}, nil
}

func postgresqlResourceAttributeKeys() []string {
	return []string{
		"service.instance.id",
		"server.address",
		"server.port",
		"postgresql.database.name",
		"postgresql.schema.name",
		"postgresql.table.name",
		"postgresql.index.name",
	}
}

func postgresServerFromDSN(dsn string) (postgresServer, error) {
	if parsedURL, err := url.Parse(dsn); err == nil && parsedURL.Host != "" {
		return postgresServerFromURL(parsedURL)
	}
	return postgresServerFromKeywordDSN(dsn)
}

func postgresServerFromURL(parsedURL *url.URL) (postgresServer, error) {
	var port int
	if parsedURL.Port() != "" {
		parsedPort, err := strconv.Atoi(parsedURL.Port())
		if err != nil {
			return postgresServer{}, fmt.Errorf("invalid port %q: %w", parsedURL.Port(), err)
		}
		port = parsedPort
	}
	return postgresServer{address: parsedURL.Hostname(), port: port}, nil
}

func postgresServerFromKeywordDSN(dsn string) (postgresServer, error) {
	values := map[string]string{}
	for _, field := range strings.Fields(dsn) {
		key, value, found := strings.Cut(field, "=")
		if !found {
			continue
		}
		values[strings.ToLower(key)] = strings.Trim(value, "'\"")
	}

	var port int
	if values["port"] != "" {
		parsedPort, err := strconv.Atoi(values["port"])
		if err != nil {
			return postgresServer{}, fmt.Errorf("invalid port %q: %w", values["port"], err)
		}
		port = parsedPort
	}
	return postgresServer{address: values["host"], port: port}, nil
}

func postgresqlDatapointStatements() []string {
	return []string{
		`set(datapoint.attributes["db.system.name"], "postgresql")`,
		`set(datapoint.attributes["postgresql.database.name"], datapoint.attributes["datname"]) where datapoint.attributes["datname"] != nil`,
		`set(datapoint.attributes["postgresql.schema.name"], datapoint.attributes["schemaname"]) where datapoint.attributes["schemaname"] != nil`,
		`set(datapoint.attributes["postgresql.table.name"], datapoint.attributes["relname"]) where datapoint.attributes["relname"] != nil`,
		`set(datapoint.attributes["state"], "live") where metric.name == "pg_stat_user_tables_n_live_tup"`,
		`set(datapoint.attributes["state"], "dead") where metric.name == "pg_stat_user_tables_n_dead_tup"`,
		`set(datapoint.attributes["operation"], "ins") where metric.name == "pg_stat_database_tup_inserted_total" or metric.name == "pg_stat_user_tables_n_tup_ins_total"`,
		`set(datapoint.attributes["operation"], "upd") where metric.name == "pg_stat_database_tup_updated_total" or metric.name == "pg_stat_user_tables_n_tup_upd_total"`,
		`set(datapoint.attributes["operation"], "del") where metric.name == "pg_stat_database_tup_deleted_total" or metric.name == "pg_stat_user_tables_n_tup_del_total"`,
		`set(datapoint.attributes["operation"], "hot_upd") where metric.name == "pg_stat_user_tables_n_tup_hot_upd_total"`,
		`set(datapoint.attributes["source"], "heap_read") where metric.name == "pg_statio_user_tables_heap_blocks_read_total"`,
		`set(datapoint.attributes["source"], "heap_hit") where metric.name == "pg_statio_user_tables_heap_blocks_hit_total"`,
		`set(datapoint.attributes["source"], "idx_read") where metric.name == "pg_statio_user_tables_idx_blocks_read_total"`,
		`set(datapoint.attributes["source"], "idx_hit") where metric.name == "pg_statio_user_tables_idx_blocks_hit_total"`,
		`set(datapoint.attributes["source"], "toast_read") where metric.name == "pg_statio_user_tables_toast_blocks_read_total"`,
		`set(datapoint.attributes["source"], "toast_hit") where metric.name == "pg_statio_user_tables_toast_blocks_hit_total"`,
		`set(datapoint.attributes["source"], "tidx_read") where metric.name == "pg_statio_user_tables_tidx_blocks_read_total"`,
		`set(datapoint.attributes["source"], "tidx_hit") where metric.name == "pg_statio_user_tables_tidx_blocks_hit_total"`,
	}
}

func postgresqlMetricStatements() []string {
	return []string{
		`set(metric.name, "postgresql.backends") where metric.name == "pg_stat_database_numbackends"`,
		`set(metric.name, "postgresql.connection.max") where metric.name == "pg_database_connection_limit" or metric.name == "pg_settings_max_connections"`,
		`set(metric.name, "postgresql.db_size") where metric.name == "pg_database_size_bytes"`,
		`set(metric.name, "postgresql.commits") where metric.name == "pg_stat_database_xact_commit_total"`,
		`set(metric.name, "postgresql.rollbacks") where metric.name == "pg_stat_database_xact_rollback_total"`,
		`set(metric.name, "postgresql.operations") where metric.name == "pg_stat_database_tup_inserted_total" or metric.name == "pg_stat_database_tup_updated_total" or metric.name == "pg_stat_database_tup_deleted_total" or metric.name == "pg_stat_user_tables_n_tup_ins_total" or metric.name == "pg_stat_user_tables_n_tup_upd_total" or metric.name == "pg_stat_user_tables_n_tup_del_total" or metric.name == "pg_stat_user_tables_n_tup_hot_upd_total"`,
		`set(metric.name, "postgresql.rows") where metric.name == "pg_stat_user_tables_n_live_tup" or metric.name == "pg_stat_user_tables_n_dead_tup"`,
		`set(metric.name, "postgresql.index.scans") where metric.name == "pg_stat_user_tables_idx_scan_total"`,
		`set(metric.name, "postgresql.index.size") where metric.name == "pg_stat_user_tables_index_size_bytes"`,
		`set(metric.name, "postgresql.table.size") where metric.name == "pg_stat_user_tables_table_size_bytes"`,
		`set(metric.name, "postgresql.table.vacuum.count") where metric.name == "pg_stat_user_tables_vacuum_count_total"`,
		`set(metric.name, "postgresql.blocks_read") where IsMatch(metric.name, "^pg_statio_user_tables_.*_blocks_(read|hit)_total$")`,
		`set(metric.name, "postgresql.deadlocks") where metric.name == "pg_stat_database_deadlocks_total"`,
		`set(metric.name, "postgresql.temp.io") where metric.name == "pg_stat_database_temp_bytes_total"`,
		`set(metric.name, "postgresql.temp_files") where metric.name == "pg_stat_database_temp_files_total"`,
		`set(metric.name, "postgresql.wal.lag") where metric.name == "pg_replication_lag_seconds"`,
		`set(metric.unit, "By") where metric.name == "postgresql.db_size" or metric.name == "postgresql.index.size" or metric.name == "postgresql.table.size" or metric.name == "postgresql.temp.io"`,
		`set(metric.unit, "s") where metric.name == "postgresql.wal.lag"`,
		`set(metric.unit, "{connections}") where metric.name == "postgresql.connection.max"`,
		`set(metric.unit, "{buffers}") where metric.name == "postgresql.blocks_read"`,
		`set(metric.unit, "{vacuum}") where metric.name == "postgresql.table.vacuum.count"`,
		`set(metric.unit, "1") where metric.name == "postgresql.backends" or metric.name == "postgresql.commits" or metric.name == "postgresql.rollbacks" or metric.name == "postgresql.operations" or metric.name == "postgresql.rows"`,
		`replace_pattern(metric.name, "^pg_", "postgresql.") where IsMatch(metric.name, "^pg_")`,
		`replace_pattern(metric.name, "_total$", "") where IsMatch(metric.name, "^postgresql\\.")`,
		`replace_pattern(metric.name, "_", ".") where IsMatch(metric.name, "^postgresql\\.") and not IsMatch(metric.name, "^postgresql\\.(db_size|blocks_read|temp_files)$")`,
	}
}
