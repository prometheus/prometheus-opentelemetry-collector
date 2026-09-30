# ECS Exporter Receiver

Receives Amazon ECS task metrics by running the Prometheus
[ecs_exporter](https://github.com/prometheus-community/ecs_exporter) as an
OpenTelemetry Collector receiver.

| Status        |           |
| ------------- |-----------|
| Stability     | [development]: metrics   |
| Distributions | [prometheus] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/prometheus/prometheus-opentelemetry-collector?query=is%3Aissue%20is%3Aopen%20label%3Areceiver%2Fecs%20&label=open&color=orange&logo=opentelemetry)](https://github.com/prometheus/prometheus-opentelemetry-collector/issues?q=is%3Aopen+is%3Aissue+label%3Areceiver%2Fecs) [![Closed issues](https://img.shields.io/github/issues-search/prometheus/prometheus-opentelemetry-collector?query=is%3Aissue%20is%3Aclosed%20label%3Areceiver%2Fecs%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/prometheus/prometheus-opentelemetry-collector/issues?q=is%3Aclosed+is%3Aissue+label%3Areceiver%2Fecs) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@ArthurSens](https://www.github.com/ArthurSens) |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development
[prometheus]:

## Configuration

The receiver reads the ECS task metadata endpoint from
`ECS_CONTAINER_METADATA_URI_V4`, which ECS automatically provides to task
containers. `scrape_interval` controls how often the embedded exporter is
polled and defaults to `30s`.

```yaml
receivers:
  ecs_exporter:
    scrape_interval: 30s
```

The collector must run as a container in the ECS task whose metrics it
collects. Add the receiver to a metrics pipeline in the collector service
configuration.
