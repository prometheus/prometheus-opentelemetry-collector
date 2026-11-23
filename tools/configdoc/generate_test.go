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

package configdoc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateTable(t *testing.T) {
	readme := "Introduction.\n" + beginMarker + `
| Key | Default | Description |
| --- | --- | --- |
| ` + "`exporter_config.http_timeout`" + ` | old | Required. HTTP timeout. |
| ` + "`exporter_config.concurrency.single_limit`" + ` | old | Nested setting. |
| ` + "`enabled`" + ` | old | Keep this description. |
| ` + "`targets`" + ` | old | Optional targets. |
| ` + "`label`" + ` | old | A label. |
` + endMarker + "\nHand-written example.\n"
	defaults := map[string]any{
		"exporter_config": map[string]any{
			"http_timeout": "20s",
			"concurrency":  map[string]any{"SingleLimit": json.Number("9007199254740993")},
		},
		"enabled": false,
		"targets": nil,
		"label":   "a|b",
	}
	for _, newline := range []string{"\n", "\r\n"} {
		input := strings.ReplaceAll(readme, "\n", newline)
		got, err := updateTable(input, defaults)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"`\"20s\"` | Required. HTTP timeout.",
			"`9007199254740993` | Nested setting.",
			"`false` | Keep this description.",
			"`null` | Optional targets.",
			"<code>&#34;a&#124;b&#34;</code>",
			"Introduction." + newline, "Hand-written example." + newline,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
		again, err := updateTable(got, defaults)
		if err != nil || again != got {
			t.Fatalf("unchanged defaults changed the table: %v", err)
		}
	}
}

func TestUpdateRejectsMismatchedFieldsWithoutWriting(t *testing.T) {
	for _, defaults := range []map[string]any{
		{"new_field": true},                  // The documented field was removed.
		{"enabled": true, "new_field": true}, // A new field needs a description.
		{"enabled": true, "ENABLED": false},  // Untagged keys must be unambiguous.
	} {
		path := filepath.Join(t.TempDir(), "README.md")
		original := beginMarker + "\n| `enabled` | `false` | Description. |\n" + endMarker
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := UpdateFile(path, defaults); err == nil {
			t.Fatal("expected mismatched configuration to fail")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != original {
			t.Fatalf("failed generation modified README: %v", err)
		}
	}
}

func TestUpdateRejectsMalformedTable(t *testing.T) {
	for _, readme := range []string{
		"no markers", beginMarker, endMarker + beginMarker,
		beginMarker + endMarker + beginMarker + endMarker,
		beginMarker + "\n| `enabled` | no description |\n" + endMarker,
		beginMarker + "\n| `enabled` | `false` | Description.\n" + endMarker,
		beginMarker + "\n| `enabled` | `false` | One. |\n| `ENABLED` | `true` | Two. |\n" + endMarker,
	} {
		if _, err := updateTable(readme, map[string]any{"enabled": true}); err == nil {
			t.Errorf("accepted malformed table: %q", readme)
		}
	}
}
