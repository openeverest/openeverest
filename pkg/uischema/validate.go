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

// Package uischema checks provider UI schemas (the `uiSchema` rendered by the
// UI generator) against the rules documented in docs/ui/ui-generator.
//
// The UI applies the same rules at runtime (ui-generator utils/toggleable);
// testdata/*.yaml is the shared contract both implementations are tested
// against (see toggleable-shared-cases.test.ts in the UI), so keep them in sync.
package uischema

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Rule identifies a schema rule; values are stable for tooling and the shared
// test cases.
type Rule string

// Rules checked by Validate; see README.md for their exact conditions.
const (
	RuleToggleableUnsafeKey    Rule = "toggleable-unsafe-key"
	RuleToggleableNoFields     Rule = "toggleable-no-fields"
	RuleToggleableNested       Rule = "toggleable-nested"
	RuleToggleableOverlap      Rule = "toggleable-overlap"
	RuleToggleableCELUnguarded Rule = "toggleable-cel-unguarded"
)

// Issue is a single problem found in a UI schema.
type Issue struct {
	Rule     Rule
	Topology string
	// Group is the dotted key path of the toggleable group: section key followed
	// by group keys, e.g. "advanced.monitoring".
	Group string
	// Field is the dotted key path of the field whose CEL rule reads the group;
	// empty for rules about the group itself.
	Field   string
	Message string
}

func (i Issue) String() string {
	if i.Field != "" {
		return fmt.Sprintf("topology %s: field %q: %s (%s)", i.Topology, i.Field, i.Message, i.Rule)
	}
	return fmt.Sprintf("topology %s: group %q: %s (%s)", i.Topology, i.Group, i.Message, i.Rule)
}

// Validate checks a provider UI schema: a map of topology name to topology
// (`{sections: {<key>: {components: ...}}}`), as stored in Provider.spec.uiSchema.
// Issues are sorted by topology, group and field.
func Validate(uiSchema map[string]any) []Issue {
	var issues []Issue
	for _, topology := range sortedKeys(uiSchema) {
		topologySchema, _ := asMap(uiSchema[topology])
		sections, ok := asMap(topologySchema["sections"])
		if !ok {
			continue
		}
		issues = append(issues, validateToggleables(topology, sections)...)
	}
	slices.SortStableFunc(issues, func(a, b Issue) int {
		return cmp.Or(
			cmp.Compare(a.Topology, b.Topology),
			cmp.Compare(a.Group, b.Group),
			cmp.Compare(a.Field, b.Field),
		)
	})
	return issues
}

const (
	uiTypeGroup         = "group"
	uiTypeHidden        = "hidden"
	groupTypeToggleable = "toggleable"
	degradedSuffix      = "; the UI renders it as a bordered group"
)

// Keys name the group's form-only switch; the UI rejects anything else.
var switchKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// toggleable is a group that the UI renders with a working switch.
type toggleable struct {
	key   string
	paths []string
}

// celField is a field carrying CEL rules, with the toggleable it sits in (if any).
type celField struct {
	key       string
	enclosing string
	exprs     []string
}

// validateToggleables mirrors resolveToggleable in the UI: switching a
// toggleable group off deletes every API path its fields write, so it must own
// those paths exclusively and must not sit inside another toggleable. CEL rules
// outside the group then see those paths as absent and must guard them.
func validateToggleables(topology string, sections map[string]any) []Issue {
	pathUsage := map[string]int{}
	for _, key := range sortedKeys(sections) {
		section, _ := asMap(sections[key])
		components, _ := asMap(section["components"])
		for _, p := range leafPaths(components) {
			pathUsage[p]++
		}
	}

	var (
		issues    []Issue
		active    []toggleable
		celFields []celField
	)

	// enclosing is the key of the active toggleable the components sit in, or "".
	var walk func(components map[string]any, parent []string, enclosing string)
	walk = func(components map[string]any, parent []string, enclosing string) {
		for _, key := range sortedKeys(components) {
			item, _ := asMap(components[key])
			keyPath := slices.Concat(parent, []string{key})
			uiType, _ := item["uiType"].(string)
			if uiType != uiTypeGroup && uiType != uiTypeHidden {
				if exprs := celExpressions(item); len(exprs) > 0 {
					celFields = append(celFields, celField{key: strings.Join(keyPath, "."), enclosing: enclosing, exprs: exprs})
				}
				continue
			}
			children, ok := asMap(item["components"])
			if !ok {
				continue
			}
			childEnclosing := enclosing
			if groupType, _ := item["groupType"].(string); uiType == uiTypeGroup && groupType == groupTypeToggleable {
				paths := leafPaths(children)
				groupKey := strings.Join(keyPath, ".")
				if rule, msg := degradeRule(keyPath, paths, enclosing != "", pathUsage); rule != "" {
					issues = append(issues, Issue{Rule: rule, Topology: topology, Group: groupKey, Message: msg + degradedSuffix})
				} else {
					childEnclosing = groupKey
					active = append(active, toggleable{key: groupKey, paths: paths})
				}
			}
			walk(children, keyPath, childEnclosing)
		}
	}
	for _, key := range sortedKeys(sections) {
		section, _ := asMap(sections[key])
		components, _ := asMap(section["components"])
		walk(components, []string{key}, "")
	}

	return append(issues, unguardedCelIssues(topology, active, celFields)...)
}

// degradeRule returns the first rule that turns a toggleable group into a
// bordered one, with its message, or "" when the group works as a toggleable.
func degradeRule(keyPath, paths []string, nested bool, pathUsage map[string]int) (Rule, string) {
	switch {
	case slices.ContainsFunc(keyPath, func(k string) bool { return !switchKeyPattern.MatchString(k) }):
		return RuleToggleableUnsafeKey, "has a section or group key outside [A-Za-z0-9_-], so it cannot name a switch"
	case len(paths) == 0:
		return RuleToggleableNoFields, "has no field with an API path, so there is nothing to switch off"
	case nested:
		return RuleToggleableNested, "is nested inside another toggleable group"
	}
	if overlap := firstSharedPath(paths, pathUsage); overlap != "" {
		return RuleToggleableOverlap, fmt.Sprintf("shares the path %q with a field outside it; switching it off would delete that field's value", overlap)
	}
	return "", ""
}

func unguardedCelIssues(topology string, active []toggleable, celFields []celField) []Issue {
	var issues []Issue
	for _, field := range celFields {
		for _, group := range active {
			if group.key == field.enclosing {
				continue
			}
			if path := firstUnguardedRef(field.exprs, group.paths); path != "" {
				issues = append(issues, Issue{
					Rule:     RuleToggleableCELUnguarded,
					Topology: topology,
					Group:    group.key,
					Field:    field.key,
					Message: fmt.Sprintf("a CEL rule reads %q of toggleable group %q without has(%s); it fails while the group is switched off",
						path, group.key, path),
				})
			}
		}
	}
	return issues
}

var (
	// A `spec.` path not preceded by an identifier or a dot, so `original.spec.x` is skipped.
	celSpecRef = regexp.MustCompile(`(^|[^\w.])(spec(?:\.\w+)+)`)
	celHasCall = regexp.MustCompile(`has\(\s*(spec(?:\.\w+)+)\s*\)`)
)

// firstUnguardedRef returns a group path that some expression reads (directly
// or below it) without a has() on that exact path, or "".
func firstUnguardedRef(exprs, groupPaths []string) string {
	for _, expr := range exprs {
		guarded := map[string]bool{}
		for _, m := range celHasCall.FindAllStringSubmatch(expr, -1) {
			guarded[m[1]] = true
		}
		for _, m := range celSpecRef.FindAllStringSubmatch(expr, -1) {
			ref := m[2]
			for _, p := range groupPaths {
				if (ref == p || strings.HasPrefix(ref, p+".")) && !guarded[p] {
					return p
				}
			}
		}
	}
	return ""
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

// firstSharedPath returns a path of the group that is also written by a field
// outside it, or "".
func firstSharedPath(groupPaths []string, topologyUsage map[string]int) string {
	own := map[string]int{}
	for _, p := range groupPaths {
		own[p]++
	}
	for _, p := range groupPaths {
		if topologyUsage[p] > own[p] {
			return p
		}
	}
	return ""
}

// leafPaths lists the API paths written by every field under components, in
// the UI's terms: groups (and `hidden` containers) are walked, hidden fields
// and items without a uiType write nothing, and `path` is a string or a list
// of strings (multi-path).
func leafPaths(components map[string]any) []string {
	var paths []string
	for _, key := range sortedKeys(components) {
		item, _ := asMap(components[key])
		uiType, _ := item["uiType"].(string)
		if uiType == uiTypeGroup || uiType == uiTypeHidden {
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
		seen := map[string]bool{}
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

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
