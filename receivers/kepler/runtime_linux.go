// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package kepler

import (
	keplercollector "github.com/sustainable-computing-io/kepler/collector"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
	"log/slog"
)

func newKeplerRuntime(cfg *keplerconfig.Config, logger *slog.Logger) (runtime, error) {
	return keplercollector.NewRuntime(cfg, logger)
}
