# Kepler Exporter Receiver

Embeds Kepler's Go runtime to collect host and workload energy/power metrics.
The receiver gathers client_golang collectors directly through the Prometheus
OpenTelemetry Collector bridge. No Kepler process or HTTP endpoint is needed.

This draft depends on ArthurSens/kepler's `embeddable-runtime` branch. The exact
commit is pinned in go.mod and the distribution builder configuration. Replace
the fork replacement with an upstream release before merging.

## Configuration

```yaml
receivers:
  kepler_exporter:
    scrape_interval: 30s
    exporter_config:
      host:
        procfs: /host/proc
        sysfs: /host/sys
      monitor:
        interval: 5s
      exporter:
        prometheus:
          metricsLevel: [node, container, pod]
          debugCollectors: []
      kube:
        enabled: true
        nodeName: ${env:K8S_NODE_NAME}

exporters:
  debug: {}

service:
  pipelines:
    metrics:
      receivers: [kepler_exporter]
      exporters: [debug]
```

`exporter_config` accepts Kepler's native YAML keys (including camelCase),
duration strings, and metrics-level lists. Missing fields use fresh upstream
defaults. Unknown fields are rejected. `scrape_interval` controls export cadence;
`monitor.interval` controls Kepler's background sampling independently.

For standalone Linux hosts, leave `kube.enabled` false (the default). Kepler can
attribute node and process energy without Kubernetes. Kubernetes mode uses
Kepler's own kubelet/API-server discovery and needs its corresponding credentials
and RBAC. It does not depend on a k8sattributes processor.

## Deployment requirements

Run one receiver per Linux host, usually in an agent DaemonSet. Provide readable
host procfs/sysfs mounts and supported RAPL or hwmon sensors. With the default
`kube.podInformer.mode: kubelet`, Kepler reads the Node object to locate the
kubelet, then polls its authenticated `/pods` endpoint. API-server informer mode
is also supported. Supply the Collector pod/container identity in Kepler's `kube`
settings when needed for self-attribution in a private cgroup namespace.

Each named receiver owns its configuration, sampling services, registry, and
shutdown function. Multiple instances are supported, though sampling the same
node twice is usually unnecessary.

Kepler's web listener, stdout, and pprof settings are CLI-only and do not create
endpoints here. Debug Go/process collectors describe the hosting Collector
process; set `debugCollectors: []` if these are not wanted.

Kepler collection requires Linux and CGO (including its optional NVML dependency).
The distribution remains buildable on other platforms; starting Kepler there is
unsupported. Run runtime tests on Linux. Fake-meter tests validate wiring and
lifecycle, not real sensor accuracy or live Kubernetes permissions.
