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

// isTemplateWidget reports whether a widget stands for a piece of schema
// (`itemTemplate`) rather than render anything itself.
func isTemplateWidget(widgetType string) bool {
	return widgetType == widgetTypeKeyValue
}

// ignoredRowFieldParams returns the params of a map row's key or value that
// have nothing to act on: a row starts empty and writes only its key and value.
func ignoredRowFieldParams() []string {
	return []string{"defaultValue", "badge", "badgeToApi"}
}

// branch is a `validation` block, or one of its mode branches, with where it
// sits, e.g. "validation.modes.edit".
type branch struct {
	at    string
	rules map[string]any
}

// lintFields mirrors lintMapFields in the UI: it reports what the UI renders
// anyway while ignoring part of it. For every map field the UI renders (not
// in droppedMaps): options it ignores, a default it can't use and limits no
// map meets; for every field: a template widget used outside an itemTemplate.
//
//	labels:                          # map-ignored-option: regex is ignored
//	  uiType: map
//	  path: spec.labels
//	  validation: {regex: {pattern: "^a"}}
//	  itemTemplate: {uiType: widget, widgetType: keyValue}
//	row:                             # template-widget-misplaced
//	  uiType: widget
//	  widgetType: keyValue
func lintFields(topology string, sections map[string]any, droppedMaps map[string]bool) []Issue {
	var issues []Issue
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		for _, f := range collectFields(components, []string{key}) {
			for _, finding := range lintField(f, droppedMaps) {
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
func lintField(f field, droppedMaps map[string]bool) []finding {
	if widgetType, _ := f.item["widgetType"].(string); f.item["uiType"] == uiTypeWidget && isTemplateWidget(widgetType) {
		return []finding{{
			rule: RuleTemplateWidgetMisplaced,
			message: fmt.Sprintf("is the %q template widget, which stands only for a map's row in itemTemplate; anywhere else it renders nothing",
				widgetType,
			),
		}}
	}
	if f.item["uiType"] != uiTypeMap || droppedMaps[f.key] {
		return nil
	}
	var findings []finding
	if ignored := ignoredMapOptions(f.item); len(ignored) > 0 {
		findings = append(findings, finding{
			rule:    RuleMapIgnoredOption,
			message: fmt.Sprintf("sets %s, which a map field ignores", strings.Join(ignored, ", ")),
		})
	}
	if defaults := invalidMapDefaults(f.item); len(defaults) > 0 {
		findings = append(findings, finding{
			rule:    RuleMapInvalidDefault,
			message: fmt.Sprintf("sets %s to something other than a map of strings, so the field starts empty", strings.Join(defaults, ", ")),
		})
	}
	if limits := describeInvalidLimits(f.item["validation"]); limits != "" {
		findings = append(findings, finding{rule: RuleMapInvalidLimits, message: limits})
	}
	return findings
}

// ignoredMapOptions lists the options of a map field the UI ignores: `regex`
// and `celExpressions` on the map, CEL on a row's key or value, and the row
// params from ignoredRowFieldParams. Mode branches count too.
//
//	validation:
//	  regex: {pattern: "^a"}                 # -> validation.regex
//	itemTemplate:
//	  uiType: widget
//	  widgetType: keyValue
//	  fieldParams:
//	    value: {defaultValue: x}             # -> itemTemplate.fieldParams.value.defaultValue
//	  validation:
//	    key:
//	      celExpressions: [{celExpr: "..."}] # -> itemTemplate.validation.key.celExpressions
func ignoredMapOptions(item map[string]any) []string {
	var ignored []string
	for _, b := range declaredBranches(item["validation"], "validation") {
		if _, ok := b.rules["regex"]; ok {
			ignored = append(ignored, b.at+".regex")
		}
		if hasCel(b.rules) {
			ignored = append(ignored, b.at+".celExpressions")
		}
	}
	itemTemplate, _ := item["itemTemplate"].(map[string]any)
	fieldParams, _ := itemTemplate["fieldParams"].(map[string]any)
	validation, _ := itemTemplate["validation"].(map[string]any)
	for _, rowField := range []string{"key", "value"} {
		params, _ := fieldParams[rowField].(map[string]any)
		for _, param := range ignoredRowFieldParams() {
			if _, ok := params[param]; ok {
				ignored = append(ignored, fmt.Sprintf("itemTemplate.fieldParams.%s.%s", rowField, param))
			}
		}
		for _, b := range declaredBranches(validation[rowField], "itemTemplate.validation."+rowField) {
			if hasCel(b.rules) {
				ignored = append(ignored, b.at+".celExpressions")
			}
		}
	}
	return ignored
}

// invalidMapDefaults lists the defaults of a map field, in fieldParams or one
// of its modes, that are not a map of strings. The UI ignores them.
func invalidMapDefaults(item map[string]any) []string {
	fieldParams, _ := item["fieldParams"].(map[string]any)
	var invalid []string
	if value, ok := fieldParams["defaultValue"]; ok && !isStringMap(value) {
		invalid = append(invalid, "fieldParams.defaultValue")
	}
	modes, _ := fieldParams["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		params, _ := modes[mode].(map[string]any)
		if value, ok := params["defaultValue"]; ok && !isStringMap(value) {
			invalid = append(invalid, fmt.Sprintf("fieldParams.modes.%s.defaultValue", mode))
		}
	}
	return invalid
}

// describeInvalidLimits returns why `minItems` / `maxItems` can't work in the
// first validation branch where they don't, or "". A mode branch takes the
// shared rules unless it sets `inheritShared: false`, as in the UI.
//
//	validation:
//	  minItems: 3
//	  maxItems: 2      # -> "validation needs at least 3 entries but allows at most 2, ..."
func describeInvalidLimits(validation any) string {
	for _, b := range effectiveBranches(validation) {
		for _, limit := range []string{"minItems", "maxItems"} {
			if value, ok := b.rules[limit]; ok && !isCount(value) {
				return fmt.Sprintf("%s.%s is not a whole number of 0 or more", b.at, limit)
			}
		}
		minItems, hasMin := b.rules["minItems"]
		maxItems, hasMax := b.rules["maxItems"]
		if hasMin && hasMax && toFloat(minItems) > toFloat(maxItems) {
			return fmt.Sprintf("%s needs at least %v entries but allows at most %v, so no map passes", b.at, minItems, maxItems)
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

func isStringMap(value any) bool {
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for _, entry := range m {
		if _, ok := entry.(string); !ok {
			return false
		}
	}
	return true
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
