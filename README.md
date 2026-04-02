# prometheus-opentelemetry-collector

[EXPERIMENTAL] A Prometheus-flavored OpenTelemetry Collector distribution built with the OpenTelemetry Collector Builder (OCB).

This repository is intentionally small. Its job is to define a curated collector build, generate the distribution during local builds and CI, and fail fast when component additions or version bumps stop compiling.

## Goals

- Add Prometheus exporters that can be embedded as native collector components.
- Verify that the generated collector still compiles whenever dependencies or component selections change.

## Repository layout

- `builder-config.yaml`: the single source of truth for the distribution
- `Makefile`: local entrypoints for generation, compile checks, and cleanup
- `.github/workflows/build.yaml`: CI that regenerates the collector and compiles the generated Go module

Generated sources and binaries live in `_build/` and are not committed.

## Local development

```bash
make generate
make build
make check
make clean
```

Command behavior:

- `make generate`: install OCB and generate collector sources into `_build/`
- `make build`: run OCB end-to-end and produce the `prometheus-otelcol` binary
- `make check`: regenerate sources, then run `go build ./...` inside `_build/`
- `make clean`: remove generated artifacts and the local OCB binary

## Adding components or bumping versions safely

1. Edit `builder-config.yaml`.
2. Run `make check` to regenerate the collector and compile the generated module.
