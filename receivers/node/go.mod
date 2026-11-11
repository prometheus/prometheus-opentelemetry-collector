module github.com/prometheus/prometheus-opentelemetry-collector/receivers/node

go 1.26.0

require (
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/prometheus/client_golang v1.25.0
	github.com/prometheus/node_exporter v0.0.0
	github.com/prometheus/opentelemetry-collector-bridge v0.2.0
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/collector/component v1.69.0
	go.opentelemetry.io/collector/component/componenttest v0.163.0
	go.opentelemetry.io/collector/confmap v1.69.0
	go.opentelemetry.io/collector/consumer/consumertest v0.163.0
	go.opentelemetry.io/collector/pdata v1.69.0
	go.opentelemetry.io/collector/receiver v1.69.0
	go.opentelemetry.io/collector/receiver/receivertest v0.163.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap/exp v0.3.0
)

require (
	cyphar.com/go-pathrs v0.2.2 // indirect
	github.com/beevik/ntp v1.6.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/dennwc/btrfs v0.0.0-20260222081608-edfb8b9e4f55 // indirect
	github.com/dennwc/ioctl v1.0.1-0.20181021180353-017804252068 // indirect
	github.com/ema/qdisc v1.0.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.2.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-envparse v0.1.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hodgesds/perf-utils v0.7.0 // indirect
	github.com/illumos/go-kstat v0.0.0-20210513183136-173c9b0a9973 // indirect
	github.com/jsimonetti/rtnetlink/v2 v2.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.3 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.1 // indirect
	github.com/knadh/koanf/v2 v2.3.8 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lufia/iostat v1.2.1 // indirect
	github.com/mattn/go-xmlrpc v0.0.3 // indirect
	github.com/mdlayher/ethtool v0.7.0 // indirect
	github.com/mdlayher/genetlink v1.4.0 // indirect
	github.com/mdlayher/netlink v1.11.2 // indirect
	github.com/mdlayher/socket v0.6.1 // indirect
	github.com/mdlayher/wifi v0.10.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/selinux v1.15.1 // indirect
	github.com/pandatix/go-abnf v0.5.3 // indirect
	github.com/power-devops/perfstat v0.0.0-20260805114148-88456608a4f6 // indirect
	github.com/prometheus-community/go-runit v0.1.0 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.73.1 // indirect
	github.com/prometheus/procfs v0.23.0 // indirect
	github.com/safchain/ethtool v0.7.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/consumer v1.69.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.163.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.163.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.69.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.163.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.163.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.69.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.163.0 // indirect
	go.opentelemetry.io/collector/receiver/receiverhelper v0.163.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.163.0 // indirect
	go.opentelemetry.io/collector/scraper v0.163.0 // indirect
	go.opentelemetry.io/collector/scraper/scraperhelper v0.163.0 // indirect
	go.opentelemetry.io/contrib/bridges/prometheus v0.72.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260803160001-6ac0973c030d // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	howett.net/plist v1.0.1 // indirect
)

// Temporary dependency on prometheus/node_exporter#3661 for embedding review.
replace github.com/prometheus/node_exporter => github.com/nicolastakashi/node_exporter v0.0.0-20261111165904-ab590e822c83
