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
	"fmt"
	"html"
	"os"
	"slices"
	"strings"
)

const (
	beginMarker = "<!-- configdoc -->"
	endMarker   = "<!-- /configdoc -->"
)

// UpdateFile refreshes README default cells from a decoded JSON configuration.
// It validates all rows before writing and preserves descriptions and examples.
func UpdateFile(path string, defaults map[string]any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := updateTable(string(content), defaults)
	if err != nil || updated == string(content) {
		return err
	}
	return os.WriteFile(path, []byte(updated), 0o644)
}

// updateTable changes only default cells. Keys, descriptions, and required-field
// notes stay in the README, rather than in a second documentation schema.
func updateTable(readme string, defaults map[string]any) (string, error) {
	if strings.Count(readme, beginMarker) != 1 || strings.Count(readme, endMarker) != 1 {
		return "", fmt.Errorf("expected one pair of configdoc markers")
	}
	start := strings.Index(readme, beginMarker) + len(beginMarker)
	end := strings.Index(readme, endMarker)
	if start > end {
		return "", fmt.Errorf("configdoc markers are out of order")
	}
	values := make(map[string]string)
	if err := flatten(defaults, "", values); err != nil {
		return "", err
	}
	newline := "\n"
	if strings.Contains(readme, "\r\n") {
		newline = "\r\n"
	}
	lines := strings.Split(readme[start:end], newline)
	seen := make(map[string]bool)
	for i, line := range lines {
		row := strings.TrimSpace(line)
		if !strings.HasPrefix(row, "| `") {
			continue
		}
		if !strings.HasSuffix(row, " |") {
			return "", fmt.Errorf("invalid configuration row: %s", line)
		}
		cells := strings.SplitN(strings.TrimSpace(row[1:len(row)-1]), " | ", 3)
		if len(cells) != 3 {
			return "", fmt.Errorf("expected Key, Default, Description columns: %s", line)
		}
		key := strings.Trim(cells[0], "`")
		name := normalize(key)
		value, ok := values[name]
		if !ok || seen[name] {
			return "", fmt.Errorf("unknown or duplicate configuration row %q", key)
		}
		seen[name] = true
		cells[1] = code(value)
		lines[i] = "| " + strings.Join(cells, " | ") + " |"
	}
	var missing []string
	for key := range values {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		return "", fmt.Errorf("add README rows with descriptions for: %s", strings.Join(missing, ", "))
	}
	return readme[:start] + strings.Join(lines, newline) + readme[end:], nil
}

func flatten(values map[string]any, prefix string, out map[string]string) error {
	for key, value := range values {
		path := prefix + key
		if nested, ok := value.(map[string]any); ok && len(nested) != 0 {
			if err := flatten(nested, path+".", out); err != nil {
				return err
			}
			continue
		}
		key = normalize(path)
		if _, exists := out[key]; exists {
			return fmt.Errorf("ambiguous configuration key %q", path)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out[key] = string(data)
	}
	return nil
}

// The two supported receivers use the bridge's untagged decoder, which matches
// Go field names without regard to underscores or case, including nested fields.
func normalize(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "_", ""))
}

func code(value string) string {
	if strings.ContainsAny(value, "`|") {
		return "<code>" + strings.ReplaceAll(html.EscapeString(value), "|", "&#124;") + "</code>"
	}
	return "`" + value + "`"
}
