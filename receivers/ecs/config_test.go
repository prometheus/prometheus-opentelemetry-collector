// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package ecs

import "testing"

func TestConfigUnmarshalerGetConfigStruct(t *testing.T) {
	t.Parallel()

	got := configUnmarshaler{}.GetConfigStruct()
	if _, ok := got.(*exporterConfig); !ok {
		t.Fatalf("GetConfigStruct() returned unexpected type %T", got)
	}
}
