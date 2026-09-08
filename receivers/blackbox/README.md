# Blackbox Exporter Receiver

Receives probe metrics by embedding the Prometheus blackbox exporter.

## Configuration

`scrape_interval` controls how often all configured targets are probed. Configure
blackbox modules directly as structured YAML under `modules`, or set
`config_file` to a standard blackbox exporter configuration file. These options
are mutually exclusive.

```yaml
receivers:
  blackbox_exporter:
    scrape_interval: 30s
    exporter_config:
      modules:
        http_2xx:
          prober: http
          timeout: 5s
      targets:
        - name: example
          address: https://example.com
          module: http_2xx
          labels:
            environment: production
      probe_timeout_offset: 500ms
      max_timeout: 10s
```

Each emitted probe metric includes `target`, `target_name`, and `module`
attributes, together with the target's configured labels. `max_timeout` caps a
probe when its module does not specify a shorter timeout;
`probe_timeout_offset` is subtracted from that cap.
