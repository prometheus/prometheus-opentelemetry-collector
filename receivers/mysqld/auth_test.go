// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");

package mysqld

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/mysqld_exporter/auth"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"github.com/prometheus/prometheus-opentelemetry-collector/receivers/mysqld/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.yaml.in/yaml/v3"
)

func TestFactory_Authentication(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		dsn      string
		provider any
		wantErr  string
	}{
		{name: "native", dsn: "exporter:secret@tcp(db.example:3306)/"},
		{name: "IAM", provider: map[string]any{"type": "aws-iam", "region": "us-east-1"}},
		{name: "IAM enables TLS", dsn: "exporter@tcp(db.example:3306)/", provider: map[string]any{"type": "aws-iam"}},
		{name: "IAM default SDK region", provider: map[string]any{"type": "aws-iam"}},
		{name: "IAM assumed role", provider: map[string]any{"type": "aws-iam", "region": "us-east-1", "role_arn": "arn:aws:iam::123456789012:role/Exporter"}},
		{name: "missing type", provider: map[string]any{}, wantErr: "unsupported auth provider type"},
		{name: "unknown type", provider: map[string]any{"type": "unknown"}, wantErr: "unsupported auth provider type"},
		{name: "unknown setting", provider: map[string]any{"type": "aws-iam", "regoin": "us-east-1"}, wantErr: "regoin"},
		{name: "wrong region type", provider: map[string]any{"type": "aws-iam", "region": 123}, wantErr: "region"},
		{name: "wrong role type", provider: map[string]any{"type": "aws-iam", "role_arn": true}, wantErr: "role_arn"},
		{name: "scalar", provider: "aws-iam", wantErr: "auth_provider must be a mapping"},
		{name: "list", provider: []any{"aws-iam"}, wantErr: "auth_provider must be a mapping"},
		{name: "password rejected", dsn: "exporter:secret@tcp(db.example:3306)/", provider: map[string]any{"type": "aws-iam"}, wantErr: "a password must not be set"},
		{name: "TLS disabled", dsn: "exporter@tcp(db.example:3306)/?tls=false", provider: map[string]any{"type": "aws-iam"}, wantErr: "TLS is required"},
		{name: "plaintext fallback", dsn: "exporter@tcp(db.example:3306)/?tls=preferred", provider: map[string]any{"type": "aws-iam"}, wantErr: "plaintext fallback is not allowed"},
		{name: "socket", dsn: "exporter@unix(/tmp/mysql.sock)/", provider: map[string]any{"type": "aws-iam"}, wantErr: "a TCP endpoint is required"},
		{name: "missing user", dsn: "tcp(db.example:3306)/", provider: map[string]any{"type": "aws-iam"}, wantErr: "a database user is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dsn := tt.dsn
			if dsn == "" {
				dsn = "exporter@tcp(db.example:3306)/?tls=true"
			}
			cfg := map[string]any{"data_source_name": dsn, "auth_provider": tt.provider}
			err := createMetrics(t, cfg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Exercise the configured provider with local test credentials: this signs a
// token without contacting AWS or MySQL, and checks the connection's endpoint,
// database user, region, and fresh-provider lifetime.
func TestAuthProviderDecodeHook_SignsForEachInstance(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_SESSION_TOKEN", "test-session")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// An explicit receiver region must override the process-wide default.
	t.Setenv("AWS_REGION", "eu-west-1")

	var previous auth.Provider
	for _, tt := range []struct{ region, endpoint, user string }{
		{"us-east-1", "first.example:3306", "first_exporter"},
		{"us-west-2", "second.example:3307", "second_exporter"},
	} {
		raw := map[string]any{"type": "aws-iam", "region": tt.region}
		decoded, err := authProviderDecodeHook(reflect.TypeOf(raw), reflect.TypeFor[auth.Provider](), raw)
		require.NoError(t, err)
		provider, ok := decoded.(auth.Provider)
		require.True(t, ok)
		require.Equal(t, auth.AWSIAM, provider.Name())
		if previous != nil {
			require.NotSame(t, previous, provider)
		}
		previous = provider

		cfg := mysql.NewConfig()
		cfg.Net, cfg.Addr, cfg.User = "tcp", tt.endpoint, tt.user
		require.NoError(t, provider.Configure(cfg))
		require.NoError(t, provider.Authenticate(t.Context(), cfg))
		token, err := url.Parse("https://" + cfg.Passwd)
		require.NoError(t, err)
		require.Equal(t, tt.endpoint, token.Host)
		require.Equal(t, tt.user, token.Query().Get("DBUser"))
		require.Contains(t, token.Query().Get("X-Amz-Credential"), "/"+tt.region+"/rds-db/aws4_request")
		require.Equal(t, "test-session", token.Query().Get("X-Amz-Security-Token"))
		require.NotEmpty(t, token.Query().Get("X-Amz-Signature"))
	}
}

func TestIAMExample(t *testing.T) {
	// Keep the documented YAML executable through the Collector's unmarshaler.
	contents, err := os.ReadFile("testdata/iam.yaml")
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &raw))
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	require.NoError(t, confmap.NewFromStringMap(raw).Unmarshal(cfg))
	_, err = factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, new(consumertest.MetricsSink))
	require.NoError(t, err)
}
