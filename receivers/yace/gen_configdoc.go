//go:build ignore

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

// Run by go generate; this file is excluded from the receiver's runtime build.
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	receiver "github.com/prometheus/prometheus-opentelemetry-collector/receivers/yace"
)

func main() {
	cfg := receiver.NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)
	var input bytes.Buffer
	if err := json.NewEncoder(&input).Encode(map[string]any{
		"scrape_interval": cfg.ScrapeInterval.String(),
		"exporter_config": cfg.ExporterConfig,
	}); err != nil {
		log.Fatal(err)
	}
	readme, err := filepath.Abs("README.md")
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "-C", "../../tools/configdoc", "run", "./cmd/configdoc", "-readme", readme)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = &input, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
