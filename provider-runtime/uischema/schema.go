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
	"maps"
	"slices"
)

// Helpers for reading the untyped UI schema, shared by all validators.

const (
	uiTypeGroup  = "group"
	uiTypeHidden = "hidden"
)

// forEachSection calls fn with each section key and its components, in key order.
func forEachSection(sections map[string]any, fn func(key string, components map[string]any)) {
	for _, key := range sortedKeys(sections) {
		section, _ := asMap(sections[key])
		components, _ := asMap(section["components"])
		fn(key, components)
	}
}

// isContainer reports whether the item holds child components instead of being a field.
func isContainer(item map[string]any) bool {
	uiType, _ := item["uiType"].(string)
	return uiType == uiTypeGroup || uiType == uiTypeHidden
}

// topologyPathUsage counts how many fields of the topology write each API path.
func topologyPathUsage(sections map[string]any) map[string]int {
	usage := make(map[string]int)
	forEachSection(sections, func(_ string, components map[string]any) {
		for _, p := range leafPaths(components) {
			usage[p]++
		}
	})
	return usage
}

// leafPaths lists the API paths written by every field under components, in
// the UI's terms: groups (and `hidden` containers) are walked, hidden fields
// and items without a uiType write nothing, and `path` is a string or a list
// of strings (multi-path).
func leafPaths(components map[string]any) []string {
	var paths []string
	for _, key := range sortedKeys(components) {
		item, _ := asMap(components[key])
		if isContainer(item) {
			if children, ok := asMap(item["components"]); ok {
				paths = append(paths, leafPaths(children)...)
			}
			continue
		}
		if _, declared := item["uiType"]; !declared {
			continue
		}
		paths = append(paths, targetPaths(item["path"])...)
	}
	return paths
}

func targetPaths(path any) []string {
	switch p := path.(type) {
	case string:
		if p != "" {
			return []string{p}
		}
	case []any:
		seen := make(map[string]bool)
		var out []string
		for _, v := range p {
			if s, ok := v.(string); ok && s != "" && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// celExpressions lists the CEL expressions of a field in every form mode:
// `validation.celExpressions` and `validation.modes.<mode>.celExpressions`.
func celExpressions(item map[string]any) []string {
	validation, _ := asMap(item["validation"])
	exprs := celExprList(validation)
	modes, _ := asMap(validation["modes"])
	for _, mode := range sortedKeys(modes) {
		branch, _ := asMap(modes[mode])
		exprs = append(exprs, celExprList(branch)...)
	}
	return exprs
}

func celExprList(validation map[string]any) []string {
	list, _ := validation["celExpressions"].([]any)
	var exprs []string
	for _, entry := range list {
		m, _ := asMap(entry)
		if expr, ok := m["celExpr"].(string); ok && expr != "" {
			exprs = append(exprs, expr)
		}
	}
	return exprs
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
