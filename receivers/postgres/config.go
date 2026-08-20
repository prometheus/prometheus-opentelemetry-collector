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
	"github.com/prometheus-community/postgres_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

var _ prombridge.ConfigUnmarshaler = configUnmarshaler{}

// exporterConfig is a distinct type over config.Config so it can carry a
// Validate() error method. config.Config.Validate returns a ValidatedConfig,
// which does not satisfy the interface the bridge asserts on to validate
// exporter config at load time. Defining the type rather than embedding keeps
// the field set identical, so the bridge decodes YAML and renders component
// defaults exactly as it would for config.Config.
type exporterConfig config.Config

func (c *exporterConfig) Validate() error {
	_, err := c.validated()
	return err
}

func (c *exporterConfig) validated() (config.ValidatedConfig, error) {
	return config.Config(*c).Validate()
}

type configUnmarshaler struct{}

func (configUnmarshaler) GetConfigStruct() any {
	cfg := exporterConfig(config.NewConfigWithDefaults())
	return &cfg
}
