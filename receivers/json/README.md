# JSON Exporter Receiver

Collects metrics from HTTP JSON APIs by embedding Prometheus Community's
`json_exporter`. It gathers the exporter's native collectors directly, without
a separate exporter process or `/probe` server.

## Configuration

`scrape_interval` controls collection frequency (default `30s`). Under
`exporter_config`, supply JSON exporter `modules` inline or use `config_file`
to load an existing JSON exporter YAML file. The two options are mutually
exclusive. Modules use the exporter's native field names and support JSONPath,
object metrics, labels, custom timestamps, missing keys, authentication, TLS,
accepted status codes, and POST body templates.

```yaml
receivers:
  json_exporter:
    scrape_interval: 30s
    exporter_config:
      timeout: 10s
      modules:
        default:
          metrics:
            - name: api_queue_depth
              path: '{.queue.depth}'
              valuetype: gauge
              help: Number of jobs waiting in the queue
      targets:
        - name: jobs
          address: https://api.example.com/status
          module: default
          labels:
            environment: production
```

To reuse a file, replace `modules` with `config_file: /etc/collector/json.yml`.
The file is loaded during configuration; changes require restarting the
receiver. Relative authentication/TLS file paths follow JSON exporter's existing
behavior and are resolved from the Collector's working directory.

Each target requires a unique `name` and an absolute HTTP(S) `address`. `module`
defaults to `default`. Optional `parameters` supplies lists of strings to POST
body templates, matching `/probe` query parameters:

```yaml
parameters:
  tenant: [example]
```

The receiver sets the `target` and `module` template parameters automatically.
Every metric carries `target`, `target_name`, and `module` attributes plus the
target's optional labels. These three names are reserved; target labels must
not collide with module metric labels.

`timeout` defaults to `30s` and must be positive. It bounds each target's HTTP
request, including response reading. Configured targets are collected by the
registry concurrently. Each receiver owns its runtimes and cancellation;
shutdown cancels that receiver's requests without stopping other instances.
Fetch failures are reported as scrape errors, with successfully collected
metrics retained by the bridge. JSONPath extraction follows the exporter's
existing behavior: failures are logged and affected samples are omitted.

## Draft dependency

This receiver requires the JSON exporter's new embedding API. Until that API is
released upstream, this draft pins a commit from ArthurSens's exporter fork in
both this module and the distribution builder. No upstream exporter PR is
required to build the draft. Replace both pins with an upstream release before
merging this receiver.
