# prometheus-opentelemetry-collector

[EXPERIMENTAL] A Prometheus-flavored OpenTelemetry Collector distribution built with the OpenTelemetry Collector Builder (OCB).

This repository is intentionally small. Its job is to define a curated collector build, generate the distribution during local builds and CI, and fail fast when component additions or version bumps stop compiling.

## Goals

- Add Prometheus exporters that can be embedded as native collector components.
- Verify that the generated collector still compiles whenever dependencies or component selections change.

## Repository layout

- `builder-config.yaml`: the single source of truth for the distribution
- `receivers/`: locally maintained integrations of Prometheus exporters as collector receivers
- `Makefile`: local entrypoints for generation, compile checks, and cleanup
- `.github/workflows/build.yaml`: CI that regenerates the collector and compiles the generated Go module
- `.github/workflows/test.yaml`: CI that tests local receiver modules

Generated sources and binaries live in `_build/` and are not committed.

## Local development

```bash
make gogenerate       # update generated receiver metadata and config documentation
make check-metadata   # verify generated metadata is fresh
make test             # test local receiver modules
make check            # generate and compile the collector distribution
make build            # build the collector binary
make clean            # remove generated artifacts
```

## Adding components or bumping versions safely

1. Edit `builder-config.yaml`.
2. For new local receivers, add `metadata.yaml`, `doc.go`, receiver README, and generated metadata files.
3. Run `make gogenerate`, `make check-metadata`, `make test`, and `make check`.

## Configuration documentation

After changing configuration or upgrading an exporter dependency, run
`make gogenerate` to refresh the Stackdriver and YACE README default columns.
Descriptions and required-field notes are maintained directly in the README;
new or removed fields need matching table rows. The small drivers in
`gen_configdoc.go` pass factory defaults to the standalone
[`tools/configdoc`](tools/configdoc) module without starting receivers.
Generation is explicit and is not part of `go build`.
