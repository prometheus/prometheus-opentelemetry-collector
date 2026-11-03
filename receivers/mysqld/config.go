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
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/prometheus/mysqld_exporter/auth"
	"github.com/prometheus/mysqld_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

var _ prombridge.ConfigUnmarshaler = configUnmarshaler{}

// configUnmarshaler hands the bridge a defaulted mysqld config.Config to decode
// receiver settings into. The bridge derives the OTel keys from the upstream
// struct and validates the populated config through config.Config.Validate.
type configUnmarshaler struct{}

func (configUnmarshaler) GetConfigStruct() any {
	cfg := config.NewConfigWithDefaults()
	return &cfg
}

// authProviderConfig is the serializable portion of an authentication provider.
// Each decode creates a provider owned by that receiver instance; the exporter
// handles TLS policy, credential caching, and token generation.
type authProviderConfig struct {
	Type    string `mapstructure:"type"`
	Region  string `mapstructure:"region"`
	RoleARN string `mapstructure:"role_arn"`
}

func authProviderDecodeHook(_ reflect.Type, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeFor[auth.Provider]() {
		return data, nil
	}
	raw, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("auth_provider must be a mapping")
	}

	var cfg authProviderConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &cfg,
		ErrorUnused: true,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(raw); err != nil {
		return nil, err
	}
	if auth.ProviderType(cfg.Type) != auth.AWSIAM {
		return nil, fmt.Errorf("unsupported auth provider type %q; supported: %s", cfg.Type, auth.AWSIAM)
	}
	return auth.NewAWSIAMAuth(cfg.Region, cfg.RoleARN), nil
}
