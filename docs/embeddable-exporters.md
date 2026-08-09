# Embeddable Prometheus Exporters

This guide is for Prometheus exporter maintainers who want their exporter to run as a native receiver in the Prometheus OpenTelemetry Collector distribution.

The exporter does not need to be rewritten as an OpenTelemetry component. Instead, it should expose reusable Go APIs for configuration and metric generation. The standalone binary can still read CLI flags and serve `/metrics`; downstream callers can build the same config from other inputs and consume metrics through a Prometheus registry or gatherer.

## Design Model

An embeddable exporter separates three responsibilities:

- `cmd/my_exporter`: CLI flags, environment variables, HTTP server setup, `/metrics`, web configuration, and process-level logging.
- `config`: User-facing configuration, defaults, validation, and conversion to lower-level runtime options.
- `collectors`: Implementation of Prometheus client_golang's Collector interface, while receiving an instance of Config for customization.

A good default layout looks like this:

```text
.
├── cmd/
│   └── my_exporter/
│       └── main.go
├── config/
│   └── config.go
└── collectors/
    ├── runtime.go
    ├── foo_collector.go
    └── bar_collector.go
```

`Runtime` and the implementations of `prometheus.Collector` live in the same `collectors` package. Collector files should be named after the part of the exporter they collect.

It's not a requirement that an implementation strictly adhere to the structure above. The structure is intended to convey the separation of concerns helpful for building an embeddable exporter.

A consumer of the exporter would be able to reference the config and collectors package without having dependencies on CLI logic.

### Config Package Contract

The config package must declare the exporter's complete user-facing configuration surface. If a user can tune behavior through the exporter binary, that knob should have a home in `config.Config`.

```go
package config

import "time"

type Config struct {
    DataSourceName    string
    MetricPrefix      string
    CollectionTimeout time.Duration
}
```

The config package must also provide defaults to ensure every consumer can share the exporter defined defaults.

```go
package config

import "time"

const (
    DefaultMetricPrefix      = "my_exporter"
    DefaultCollectionTimeout = 10 * time.Second
)

func NewConfigWithDefaults() Config {
    return Config{
        MetricPrefix:      DefaultMetricPrefix,
        CollectionTimeout: DefaultCollectionTimeout,
    }
}
```

Finally, the config package must validate constructed configs.

```go
package config

import "fmt"

func (c Config) Validate() error {
    if c.MetricPrefix == "" {
        return fmt.Errorf("metric prefix must not be empty")
    }
    if c.DataSourceName == "" {
        return fmt.Errorf("data source name must not be empty")
    }
    if c.CollectionTimeout <= 0 {
        return fmt.Errorf("collection timeout must be greater than zero")
    }
    return nil
}
```

With this shape, the command package becomes a thin adapter from CLI flags into `config.Config`. Similarly, the OTel Collector implementation can stay in sync with the exporter for config creation and validation.

```go
cfg := config.NewConfigWithDefaults()
cfg.DataSourceName = *dataSourceName // *dataSourceName can come from kingpin, YAML decoding or any other config strategy.

if err := cfg.Validate(); err != nil {
    return err
}
```

### Runtime Requires Validated Config

Runtime constructors should not silently assume that a config has gone through the exporter's validation path. Let `Validate()` mark successful validation and have `NewRuntime` fail clearly when that mark is missing.

```go
package config

type Config struct {
    MetricPrefix string

    validated bool
}

func (c *Config) Validate() error {
    if c.MetricPrefix == "" {
        return fmt.Errorf("metric prefix must not be empty")
    }

    c.validated = true
    return nil
}

func (c *Config) Validated() bool {
    return c.validated
}
```

```go
package collectors

func NewRuntime(cfg config.Config, logger *slog.Logger) (*Runtime, error) {
    if !cfg.Validated() {
        return nil, fmt.Errorf("config has not been validated; call cfg.Validate before NewRuntime")
        // optionally a panic would also work since this is clearly a bug.
    }

    // Resolve exporter dependencies here.
}
```

If using this pattern, `Validate()` should be a pointer method and set the marker only after all checks pass. Pass the validated config to `NewRuntime` by value so the runtime receives its own copy. `NewRuntime` should return a clear error telling callers to call `Validate()` first. Keep the marker private; expose only a narrow `Validated()` method if another package needs to check it.

This marker is a lifecycle guard, not an immutability guarantee. It is up to exporter maintainers to ensure immutability of the config between validation and runtime construction. A value copy prevents later changes to top-level scalar fields from affecting the runtime, but maps, slices, pointers, and other reference-bearing fields can still share mutable state. Copy or normalize those fields into private runtime state before using them concurrently.

### Collectors Package Contract

The collectors package must have a struct and corresponding constructor that represents a single instance of that exporter. While the traditional exporter CLI runs only one instance, when embedded, several instances of the same exporter might run as part of the same binary.

Stateful metrics exposed to provide visibility on the exporter instance itself must be fields inside this struct.

```go
// package collectors
type Runtime struct {
    metrics *Metrics
    // exporter clients, config, logger, caches, etc.
}

type Metrics struct {
    counter   prometheus.Counter
    gauge     prometheus.Gauge
    histogram prometheus.Histogram
    summary   prometheus.Summary
}

func NewRuntime(cfg config.Config, logger *slog.Logger) (*Runtime, error) {
    // Resolve exporter dependencies here.
    return &Runtime{
        // other fields
        metrics: &Metrics{
            // all metrics
        },
    }, nil
}
```

In Prometheus client_golang, a [Collector](https://github.com/prometheus/client_golang/blob/c9d5bc4c50a9b0e54f032440064a4a464333a421/prometheus/collector.go#L16-L63) is an interface implemented by several objects in that SDK. The traditional metric types (`NewCounter`, `NewGauge`, `NewHistogram`, `NewSummary`) and their `Vec` representations all implement the Collector interface. After registering collectors with a `prometheus.Registry`, their `Collect()` methods are triggered when the registry gathers metrics.

The Runtime object must have a method `Collectors() []prometheus.Collector` that returns all the collectors of a single Runtime. A returned collector can be a custom collector, a single metric, or any other type that implements `prometheus.Collector`.

```go
func (r *Runtime) Collectors() []prometheus.Collector {
    return []prometheus.Collector{
        r.metrics.counter,
        r.metrics.gauge,
        r.metrics.histogram,
        r.metrics.summary,
    }
}
```

Consumers can register that set in a registry:

```go
func (r *Runtime) Registry() (*prometheus.Registry, error) {
    registry := prometheus.NewRegistry()
    for _, collector := range r.Collectors() {
        if err := registry.Register(collector); err != nil {
            return nil, err
        }
    }
    return registry, nil
}
```

The `cmd/my_exporter` package can expose the registry with `promhttp.HandlerFor`, while downstream projects can use [prometheus-collector-bridge](https://github.com/prometheus/opentelemetry-collector-bridge) to adapt a `prometheus.Registry` into the OpenTelemetry Collector interfaces.

### Context and Shutdown

Some collectors perform database queries or call remote APIs from `Collect()`. Those operations should be cancellable so an exporter can stop promptly.

The context passed to an [OpenTelemetry component's `Start()` method](https://pkg.go.dev/go.opentelemetry.io/collector/component#Component) is intended for startup work and should not be retained for later scrapes. Instead, create a component-lifetime context and cancel function while starting the exporter, pass that context through the Runtime to its collectors, and cancel it from `Shutdown()`. Collectors should pass the context into every operation that supports cancellation.

```go
type Runtime struct {
    cancel context.CancelFunc
    foo    *FooCollector
}

func NewRuntime(cfg config.Config, logger *slog.Logger) (*Runtime, error) {
    ctx, cancel := context.WithCancel(context.Background())
    return &Runtime{
        cancel: cancel,
        foo:    NewFooCollector(ctx, cfg, logger),
    }, nil
}

func (r *Runtime) Shutdown(context.Context) error {
    r.cancel()
    return nil
}

func (r *Runtime) Collectors() []prometheus.Collector {
    return []prometheus.Collector{r.foo}
}
```

An `ExporterLifecycleManager` embedding this Runtime should call `Runtime.Shutdown` from its own `Shutdown` method. The context received by `Shutdown` bounds shutdown work; it is not the context previously supplied to in-flight operations. Calling the Runtime's cancel function signals those operations to stop.

The standard Prometheus interfaces do not carry a per-scrape context: `prometheus.Gatherer.Gather()` and `prometheus.Collector.Collect()` have no context parameter. Consequently, cancellation of an HTTP request or an OpenTelemetry scrape does not automatically reach work performed by a collector. Supporting cancellation for each individual scrape requires an exporter-specific context-aware API or another context propagation mechanism. The component-lifetime pattern above still ensures that `Shutdown()` can cancel in-flight work.

An exporter that doesn't support cancelation of its gathering methods doesn't need to worry about this section.

## Anti-patterns and Preferred Designs

### Leaking CLI logic into downstream packages

Avoid passing parsed CLI flags directly into Runtime constructors. Build a Config struct and validate it before constructing the Runtime.

Bad:

```go
var (
    dataSourceName = kingpin.Flag("data-source-name", "Postgres DSN").String()
    metricPrefix   = kingpin.Flag("metric-prefix", "Metric prefix").Default("pg").String()
)

func NewRuntime() (*Runtime, error) {
    return newRuntime(*dataSourceName, *metricPrefix)
}

func newRuntime(datasourceName, metricPrefix string) (*Runtime, error) {
    if c.MetricPrefix == "" {
        return fmt.Errorf("metric prefix must not be empty")
    }
    // rest of the code
}
```

Good:

```go
package config

type Config struct {
    DataSourceNames []string
    MetricPrefix    string
}

func NewConfigWithDefaults() Config {
    return Config{MetricPrefix: "pg"}
}

func (c Config) Validate() error {
    if c.MetricPrefix == "" {
        return fmt.Errorf("metric prefix must not be empty")
    }
    return nil
}
```

The binary can still use flags, but flags should construct the reusable configuration.

```go
var dataSourceNames = kingpin.Flag("data-source-name", "Postgres DSN").
    Required().
    Strings()

_, err := kingpin.Parse(os.Args[1:])
if err != nil {
    return err
}

cfg := config.NewConfigWithDefaults()
cfg.DataSourceNames = *dataSourceNames
if err := cfg.Validate(); err != nil {
    return err
}

runtime, err := collectors.NewRuntime(cfg, logger)
if err != nil {
    return err
}
```

### HTTP Handler as the Only Metrics Entrypoint

Avoid hiding metric generation behind an HTTP handler as the only public API. `promhttp.HandlerFor` accepts a `prometheus.Gatherer`, so the reusable boundary should be the gatherer or registry, not the HTTP handler itself.

Bad:

```go
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
    registry := prometheus.NewRegistry()
    registry.MustRegister(NewCollector(r))
    promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
```

Good:

```go
// package collectors
type Runtime struct {
    // exporter clients, config, logger, caches, etc.
}

func NewRuntime(cfg config.Config, logger *slog.Logger) (*Runtime, error) {
    // Resolve exporter dependencies here.
}

func (r *Runtime) Collectors() []prometheus.Collector {
    // Build collectors from exporter config and dependencies.
}

// cmd/my_exporter
func MetricsHandler(runtime *collectors.Runtime, logger *slog.Logger) (http.Handler, error) {
    registry := prometheus.NewRegistry()

    for _, c := range runtime.Collectors() {
        if err := registry.Register(c); err != nil {
            return nil, err
        }
    }

    opts := promhttp.HandlerOpts{
        ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
    }
    return promhttp.HandlerFor(registry, opts), nil
}
```

The bad example makes the HTTP handler the only reusable entrypoint. A collector receiver has to duplicate request handling or scrape the HTTP endpoint instead of gathering directly. The good example keeps collector construction in the reusable package and lets the binary adapt that collector set to HTTP with `promhttp.HandlerFor`.

### Global and Package Variables

Avoid declaring metric instances as package variables and/or pre-registering them in the global `prometheus.DefaultRegisterer`. Remember that, while in the CLI there's only one instance of an exporter running, if the exporter is embedded as a Go library, it is very likely that multiple instances of the same exporter are running. Global metric instances create shared state and global registration creates collisions in this case.

Add a typed struct to your runtime, and create the metrics during Runtime initialization.

Bad:

```go
var reloads = promauto.NewCounter(prometheus.CounterOpts{
    Name: "exporter_config_reloads_total",
    Help: "Total number of config reloads.",
})

type Runtime struct{}

func (r *Runtime) ReloadConfig() {
    reloads.Inc()
}
runtimeA := NewRuntime()
runtimeB := NewRuntime()
runtimeA.ReloadConfig()
runtimeB.ReloadConfig()
// Two instances incrementing the same metric.
```

Bad:

```go
type Runtime struct {
    reloads prometheus.Counter
}

func NewRuntime() *Runtime {
    reloads := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "exporter_config_reloads_total",
        Help: "Total number of config reloads.",
    })
    prometheus.MustRegister(reloads)
    return &Runtime{reloads: reloads}
}

runtimeA := NewRuntime()
runtimeB := NewRuntime()
// Two instances sharing the same prometheus.Registry.
```

Good:

```go
type Runtime struct {
    metrics *Metrics
}

type Metrics struct {
    reloads prometheus.Counter
}

func NewRuntime() (*Runtime, error) {
    return &Runtime{
        metrics: newMetrics(),
    }, nil
}

func newMetrics() *Metrics {
    reloads := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "exporter_config_reloads_total",
        Help: "Total number of config reloads.",
    })
    return &Metrics{reloads: reloads}
}

func (r *Runtime) Collectors() []prometheus.Collector {
    return []prometheus.Collector{
        r.metrics.reloads,
    }
}

func (r *Runtime) ReloadConfig() {
    r.metrics.reloads.Inc()
}
```

Package-level `*prometheus.Desc` values are also safe when they contain only metadata shared by every exporter instance. Descriptors do not hold metric sample state and are immutable after construction. Keep descriptors on the collector instance when their names, help text, or constant labels depend on configuration for that instance.

```go
var reloadsDesc = prometheus.NewDesc(
    "exporter_config_reloads_total",
    "Total number of config reloads.",
    nil,
    nil,
)

type ReloadCollector struct {
    reloads atomic.Uint64
}

func (c *ReloadCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- reloadsDesc
}

func (c *ReloadCollector) Collect(ch chan<- prometheus.Metric) {
    ch <- prometheus.MustNewConstMetric(
        reloadsDesc,
        prometheus.CounterValue,
        float64(c.reloads.Load()),
    )
}
```

In `cmd/my_exporter`, one can register the collectors with `prometheus.DefaultRegisterer` or a fresh registry. Embedded callers can register the same collectors with a scoped registry.