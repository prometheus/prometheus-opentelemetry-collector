// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mysqld

import (
	"context"
	"testing"

	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/mysqld/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type wrongConfig struct{}

func (wrongConfig) Validate() error { return nil }

func TestLifecycleManager_Start_RejectsWrongConfigType(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()

	_, err := mgr.Start(context.Background(), receivertest.NewNopSettings(metadata.Type), wrongConfig{})
	if err == nil {
		t.Fatal("Start() expected error for wrong config type, got nil")
	}
}

func TestLifecycleManager_Shutdown(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()
	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
