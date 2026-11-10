// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package node

import (
	"context"
	"testing"

	"github.com/prometheus/node_exporter/config"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/node/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestLifecycleRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	manager := newLifecycleManager()
	settings := receivertest.NewNopSettings(metadata.Type)
	for _, cfg := range []any{struct{}{}, (*config.Config)(nil)} {
		_, _, err := manager.Start(context.Background(), settings, cfg)
		require.Error(t, err)
	}
	cfg := config.NewConfigWithDefaults()
	cfg.Collectors = map[string]bool{"does_not_exist": true}
	_, _, err := manager.Start(context.Background(), settings, &cfg)
	require.ErrorContains(t, err, "unknown collector")
}
