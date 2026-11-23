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

// Command configdoc updates README configuration defaults from JSON on stdin.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/prometheus/prometheus-opentelemetry-collector/tools/configdoc"
)

func main() {
	readme := flag.String("readme", "README.md", "Markdown file containing the configuration table")
	flag.Parse()
	var defaults map[string]any
	decoder := json.NewDecoder(os.Stdin)
	decoder.UseNumber()
	if err := decoder.Decode(&defaults); err != nil {
		log.Fatal(err)
	}
	if err := configdoc.UpdateFile(*readme, defaults); err != nil {
		log.Fatal(err)
	}
}
