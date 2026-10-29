# Changelog

<!-- next version -->

## v0.3.0

### New Components

- `receiver/ecs`: Add an ecs_exporter receiver for collecting Amazon ECS task metrics. (#68)
- `receiver/mysqld`: Add the MySQL receiver for collecting MySQL server metrics through the embedded mysqld_exporter. (#16)

### Enhancements

- `receiver/yace`: Add the disable_account_alias_lookup option to skip IAM account alias lookups when collecting CloudWatch metrics. (#69)

  The option defaults to false. When enabled, the account_alias label is left empty.

### Bug Fixes

- `all`: Convert untyped Prometheus metrics to OTLP gauges instead of dropping them as unsupported. (#74)

  Provided by the OpenTelemetry Prometheus bridge dependency update to v0.72.0.

## v0.2.0

### New Components

- `receiver/blackbox`: Add the Blackbox receiver for collecting probe metrics through the embedded blackbox_exporter. (#54)
- `receiver/postgres`: Add the Postgres receiver for collecting PostgreSQL metrics through the embedded postgres_exporter. (#11)

## v0.1.0

### New Components

- `receiver/stackdriver`: Add the Stackdriver receiver for collecting Google Cloud Monitoring metrics through the embedded stackdriver_exporter. (#6)
- `receiver/yace`: Add the YACE receiver for collecting AWS CloudWatch metrics through the embedded yet-another-cloudwatch-exporter. (#10)
