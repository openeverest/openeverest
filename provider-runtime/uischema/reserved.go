// Copyright (C) 2026 The OpenEverest Contributors
//
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

package uischema

import (
	"fmt"
	"slices"
	"strings"
)

// reservedKeyPrefix starts the keys the UI engine sets on a component it
// derived, e.g. `_normalized` or `_toggleable`; a schema can't set them.
const reservedKeyPrefix = "_"

// validateReservedKeys mirrors stripEngineKeys in the UI, which drops every
// key starting with "_" from a component (a field, group or any other item
// under `components`) before reading the schema. It reports each component
// that sets one.
//
//	labels:                  # reserved-key: "_row" is the engine's
//	  uiType: map
//	  path: spec.labels
//	  _row: {...}
func validateReservedKeys(topology string, sections map[string]any) []Issue {
	issues := make([]Issue, 0, len(sections))
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		issues = append(issues, findReservedKeys(topology, components, []string{key})...)
	}
	return issues
}

func findReservedKeys(topology string, components map[string]any, parent []string) []Issue {
	var issues []Issue
	for _, key := range sortedKeys(components) {
		item, ok := components[key].(map[string]any)
		if !ok {
			continue
		}
		keyPath := slices.Concat(parent, []string{key})
		var reserved []string
		for _, itemKey := range sortedKeys(item) {
			if strings.HasPrefix(itemKey, reservedKeyPrefix) {
				reserved = append(reserved, fmt.Sprintf("%q", itemKey))
			}
		}
		if len(reserved) > 0 {
			issues = append(issues, Issue{
				Rule:     RuleReservedKey,
				Topology: topology,
				Field:    strings.Join(keyPath, "."),
				Message: fmt.Sprintf("sets %s; keys starting with %q are reserved for the UI engine, so the UI ignores them",
					strings.Join(reserved, ", "), reservedKeyPrefix,
				),
			})
		}
		if children, ok := item["components"].(map[string]any); ok {
			issues = append(issues, findReservedKeys(topology, children, keyPath)...)
		}
	}
	return issues
}
