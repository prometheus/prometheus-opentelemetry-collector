// Copyright The Prometheus Authors
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

package postgres

import (
	"github.com/go-viper/mapstructure/v2"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

var receiverType = component.MustNewType("postgres_exporter")

func NewFactory() receiver.Factory {
	return newFactoryWithLifecycleManager(newLifecycleManager())
}

func newFactoryWithLifecycleManager(lifecycleManager prombridge.ExporterLifecycleManager) receiver.Factory {
	return prombridge.NewFactoryWithUntaggedConfig(
		receiverType,
		lifecycleManager,
		configUnmarshaler{},
		prombridge.WithDecodeHooks(mapstructure.StringToTimeDurationHookFunc()),
	)
}
