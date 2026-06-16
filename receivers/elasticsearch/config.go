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

package elasticsearch

import (
	"github.com/prometheus-community/elasticsearch_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

var _ prombridge.ConfigUnmarshaler = configUnmarshaler{}

// configUnmarshaler hands the bridge a defaulted elasticsearch config.Config to
// decode into. The upstream config uses a pointer Validate method that marks the
// config as validated, so this must return *config.Config.
type configUnmarshaler struct{}

func (configUnmarshaler) GetConfigStruct() any {
	cfg := config.NewConfigWithDefaults()
	return &cfg
}
