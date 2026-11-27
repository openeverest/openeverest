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
	"maps"
	"math"
	"strings"
)

// ignoredRowFieldParams returns the params of a row's field, in a map or a
// list, that a row can't act on yet: it starts empty and writes only what its
// fields hold.
func ignoredRowFieldParams() []string {
	return []string{"defaultValue", "badge", "badgeToApi"}
}

// ignoredRowFieldOptions returns the options of a row's field that a row
// can't act on yet: it takes no options from an API, and is the same field in
// every form mode.
func ignoredRowFieldOptions() []string {
	return []string{"dataSource", "modes"}
}

// branch is a `validation` block, or one of its mode branches, with where it
// sits, e.g. "validation.modes.edit".
type branch struct {
	at    string
	rules map[string]any
}

// lintFields mirrors lintRowsFields in the UI: it reports what the UI renders
// anyway while ignoring part of it. For every map and list the UI renders
// (not in dropped): options it ignores, a default it can't use and limits no
// value meets.
//
//	labels:                          # map-ignored-option: regex is ignored
//	  uiType: map
//	  path: spec.labels
//	  validation: {regex: {pattern: "^a"}}
//	  itemTemplate: {...}
func lintFields(topology string, sections map[string]any, dropped map[string]bool) []Issue {
	var issues []Issue
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		for _, f := range collectFields(components, []string{key}) {
			for _, finding := range lintField(f, dropped) {
				issues = append(issues, Issue{
					Rule:     finding.rule,
					Topology: topology,
					Field:    f.key,
					Message:  finding.message,
				})
			}
		}
	}
	return issues
}

type finding struct {
	rule    Rule
	message string
}

// lintField returns a field's findings in the order the UI reports them.
func lintField(f field, dropped map[string]bool) []finding {
	kind, ok := rowsKindOf(f.item["uiType"])
	if !ok || dropped[f.key] {
		return nil
	}
	var findings []finding
	if ignored := ignoredRowsOptions(f.item, kind); len(ignored) > 0 {
		findings = append(findings, finding{
			rule:    kind.rules.ignoredOption,
			message: fmt.Sprintf("sets %s, which a %s field doesn't support yet and ignores", strings.Join(ignored, ", "), kind.uiType),
		})
	}
	if defaults := invalidDefaults(f.item, kind); len(defaults) > 0 {
		findings = append(findings, finding{
			rule:    kind.rules.invalidDefault,
			message: fmt.Sprintf("sets %s to something other than a %s, so the field starts empty", strings.Join(defaults, ", "), kind.valueShape),
		})
	}
	if limits := describeInvalidLimits(f.item["validation"], kind); limits != "" {
		findings = append(findings, finding{rule: kind.rules.invalidLimits, message: limits})
	}
	return findings
}

// ignoredRowsOptions lists the options of a map or a list the UI ignores:
// `regex` on the field, and what a row's field sets that a row can't act on
// yet. Mode branches count too.
//
//	validation:
//	  regex: {pattern: "^a"}                   # -> validation.regex
//	itemTemplate:                              # of a map; a list declares the field itself
//	  uiType: group
//	  components:
//	    value:
//	      uiType: text
//	      path: value
//	      fieldParams: {defaultValue: x}       # -> itemTemplate.components.value.fieldParams.defaultValue
//	      validation:
//	        celExpressions: [{celExpr: "..."}] # -> itemTemplate.components.value.validation.celExpressions
func ignoredRowsOptions(item map[string]any, kind rowsKind) []string {
	var ignored []string
	for _, b := range declaredBranches(item["validation"], "validation") {
		if _, ok := b.rules["regex"]; ok {
			ignored = append(ignored, b.at+".regex")
		}
	}
	for _, f := range kind.rowFields(item["itemTemplate"]) {
		ignored = append(ignored, ignoredRowFieldSettings(f.at, f.item)...)
	}
	return ignored
}

// ignoredRowFieldSettings lists what one row field, sitting at `at`, sets
// that a row can't act on yet. A default counts in a mode too.
func ignoredRowFieldSettings(at string, rowField map[string]any) []string {
	var ignored []string
	for _, option := range ignoredRowFieldOptions() {
		if _, ok := rowField[option]; ok {
			ignored = append(ignored, at+"."+option)
		}
	}
	fieldParams, _ := rowField["fieldParams"].(map[string]any)
	for _, param := range ignoredRowFieldParams() {
		if _, ok := fieldParams[param]; ok {
			ignored = append(ignored, fmt.Sprintf("%s.fieldParams.%s", at, param))
		}
	}
	modes, _ := fieldParams["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		params, _ := modes[mode].(map[string]any)
		if _, ok := params["defaultValue"]; ok {
			ignored = append(ignored, fmt.Sprintf("%s.fieldParams.modes.%s.defaultValue", at, mode))
		}
	}
	for _, b := range declaredBranches(rowField["validation"], at+".validation") {
		if hasCel(b.rules) {
			ignored = append(ignored, b.at+".celExpressions")
		}
	}
	return ignored
}

// invalidDefaults lists the defaults of a map or a list, in fieldParams or
// one of its modes, that are not in the shape the field writes. The UI
// ignores them.
func invalidDefaults(item map[string]any, kind rowsKind) []string {
	fieldParams, _ := item["fieldParams"].(map[string]any)
	var invalid []string
	if value, ok := fieldParams["defaultValue"]; ok && !kind.isValidDefault(value) {
		invalid = append(invalid, "fieldParams.defaultValue")
	}
	modes, _ := fieldParams["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		params, _ := modes[mode].(map[string]any)
		if value, ok := params["defaultValue"]; ok && !kind.isValidDefault(value) {
			invalid = append(invalid, fmt.Sprintf("fieldParams.modes.%s.defaultValue", mode))
		}
	}
	return invalid
}

// describeInvalidLimits returns why `min` / `max` can't work in the
// first validation branch where they don't, or "". A mode branch takes the
// shared rules unless it sets `inheritShared: false`, as in the UI.
//
//	validation:
//	  min: 3
//	  max: 2      # -> "validation needs at least 3 entries but allows at most 2, ..."
func describeInvalidLimits(validation any, kind rowsKind) string {
	for _, b := range effectiveBranches(validation) {
		for _, limit := range []string{"min", "max"} {
			if value, ok := b.rules[limit]; ok && !isCount(value) {
				return fmt.Sprintf("%s.%s is not a whole number of 0 or more", b.at, limit)
			}
		}
		minEntries, hasMin := b.rules["min"]
		maxEntries, hasMax := b.rules["max"]
		if hasMin && hasMax && toFloat(minEntries) > toFloat(maxEntries) {
			return fmt.Sprintf("%s needs at least %v entries but allows at most %v, so no %s passes", b.at, minEntries, maxEntries, kind.uiType)
		}
	}
	return ""
}

// declaredBranches returns a validation block and each of its mode branches,
// as written.
func declaredBranches(validation any, at string) []branch {
	rules, ok := validation.(map[string]any)
	if !ok {
		return nil
	}
	branches := []branch{{at: at, rules: rules}}
	modes, _ := rules["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		if modeRules, ok := modes[mode].(map[string]any); ok {
			branches = append(branches, branch{at: fmt.Sprintf("%s.modes.%s", at, mode), rules: modeRules})
		}
	}
	return branches
}

// effectiveBranches returns the rules each form mode applies: the shared ones,
// then each mode branch over them unless it sets `inheritShared: false`.
func effectiveBranches(validation any) []branch {
	rules, ok := validation.(map[string]any)
	if !ok {
		return nil
	}
	shared := maps.Clone(rules)
	delete(shared, "modes")
	branches := []branch{{at: "validation", rules: shared}}
	modes, _ := rules["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		modeRules, ok := modes[mode].(map[string]any)
		if !ok {
			continue
		}
		effective := modeRules
		if inherit, ok := modeRules["inheritShared"].(bool); !ok || inherit {
			effective = maps.Clone(shared)
			maps.Copy(effective, modeRules)
		}
		branches = append(branches, branch{at: "validation.modes." + mode, rules: effective})
	}
	return branches
}

func hasCel(rules map[string]any) bool {
	list, ok := rules["celExpressions"].([]any)
	return ok && len(list) > 0
}

// isCount reports whether value is a whole number of 0 or more. A schema
// decoded from YAML or JSON holds numbers as float64.
func isCount(value any) bool {
	switch n := value.(type) {
	case float64:
		return n >= 0 && n == math.Trunc(n)
	case int:
		return n >= 0
	case int64:
		return n >= 0
	}
	return false
}

func toFloat(value any) float64 {
	switch n := value.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}
