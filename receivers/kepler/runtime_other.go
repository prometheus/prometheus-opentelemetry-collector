// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

//go:build !linux

package kepler

import (
	"errors"
	keplerconfig "github.com/sustainable-computing-io/kepler/config"
	"log/slog"
)

func newKeplerRuntime(*keplerconfig.Config, *slog.Logger) (runtime, error) {
	return nil, errors.New("Kepler energy collection requires Linux")
}
