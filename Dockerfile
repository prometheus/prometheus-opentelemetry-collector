# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.8

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS builder
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH

COPY . .

# Build the OCB executable for the build platform first, then use it to
# generate and compile the collector for the requested target platform.
RUN make /src/.bin/builder
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-$(go env GOARCH)} make build

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /src/_build/prometheus-otelcol /prometheus-otelcol

USER nonroot:nonroot
ENTRYPOINT ["/prometheus-otelcol"]
