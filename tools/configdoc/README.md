# configdoc

A standalone Go module for refreshing configuration defaults in Markdown tables.
It uses only the Go standard library. `generate.go` contains the update logic;
`cmd/configdoc` is the command-line adapter.

Build and test from this directory:

```sh
go build ./...
go test ./...
```

Pass a JSON configuration object on stdin and select the target Markdown file:

```sh
go run ./cmd/configdoc -readme path/to/README.md < defaults.json
```

The target table must be between `<!-- configdoc -->` and `<!-- /configdoc -->`
markers, with `Key`, `Default`, and `Description` columns. Dotted keys address
nested fields. The tool updates defaults, preserves descriptions and examples,
and reports missing or stale rows before writing. It does not infer descriptions
or required-field conditions from default values.

This repository's Stackdriver and YACE generation scripts provide actual factory
defaults on stdin. Run `make gogenerate` from the repository root to update them.
