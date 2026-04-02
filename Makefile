.PHONY: generate build check clean

OCB_VERSION ?= v0.149.0
MANIFEST := builder-config.yaml
BUILD_DIR := _build
BUILDER_BIN := $(CURDIR)/.bin/builder

$(BUILDER_BIN):
	mkdir -p "$(dir $(BUILDER_BIN))"
	GOBIN="$(CURDIR)/.bin" go install go.opentelemetry.io/collector/cmd/builder@$(OCB_VERSION)

generate: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --skip-compilation --config "$(MANIFEST)"

build: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --config "$(MANIFEST)"

check: generate
	cd "$(BUILD_DIR)" && go build ./...

clean:
	rm -rf "$(BUILD_DIR)" "$(CURDIR)/.bin"
