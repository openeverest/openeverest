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

// Input is what Validate checks; new inputs are optional fields, skipped when unset.
type Input struct {
	// UISchema maps topology name to topology
	// (`{sections: {<key>: {components: ...}}}`), as stored in Provider.spec.uiSchema.
	UISchema map[string]any
}

// Validate checks a provider UI schema. Returned Issues are sorted by topology,
// group and field.
//
// in.UISchema looks like:
//
//	replicaset:              # topology
//	  sections:
//	    advanced:            # section key
//	      components:        # walked by validateToggleables
//	        monitoring: {...}
//	sharded:
//	  sections: {...}
func Validate(in Input) []Issue {
	var issues []Issue
	for _, topology := range sortedKeys(in.UISchema) {
		topologySchema, _ := asMap(in.UISchema[topology])
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

// isUnsafeKey reports whether a section or group key cannot name a switch.
func isUnsafeKey(key string) bool {
	return !switchKeyPattern.MatchString(key)
}

// toggleable is a group that the UI renders with a working switch.
type toggleable struct {
	key   string
	paths []string
}

// celField is a field carrying CEL rules, with the toggleable it sits in (if any).
//
//	advanced:
//	  components:
//	    monitoring:
//	      uiType: group
//	      groupType: toggleable
//	      components:
//	        url:
//	          uiType: text
//	          path: spec.monitoring.url
//	          validation:
//	            celExpressions:
//	              - celExpr: "spec.monitoring.url != ''"
//
// captures:
//
//	celField{
//		key:              "advanced.monitoring.url",
//		parentToggleable: "advanced.monitoring", // "" when not inside a toggleable group
//		exprs:            []string{"spec.monitoring.url != ''"},
//	}
type celField struct {
	key              string
	parentToggleable string
	exprs            []string
}

// validateToggleables mirrors resolveToggleable in the UI: switching a
// toggleable group off omits every API path its fields write, so it must own
// those paths exclusively and must not sit inside another toggleable. CEL rules
// outside the group then see those paths as absent and must guard them.
//
// sections is one topology's `sections` map:
//
//	advanced:                          # section key
//	  components:
//	    monitoring:                    # Group "advanced.monitoring"
//	      uiType: group
//	      groupType: toggleable        # toggleable group
//	      components:
//	        url:                       # leaf field, path collected by leafPaths
//	          uiType: text
//	          path: spec.monitoring.url
//	    replicas:                      # field with CEL, collected as celField
//	      uiType: number
//	      path: spec.replicas
//	      validation:
//	        celExpressions:
//	          - celExpr: "!has(spec.monitoring.url) || spec.replicas > 1"
func validateToggleables(topology string, sections map[string]any) []Issue {
	// pathWriters counts the fields writing each path across the topology.
	pathWriters := make(map[string]int)
	for _, key := range sortedKeys(sections) {
		section, _ := asMap(sections[key])
		components, _ := asMap(section["components"])
		for _, p := range leafPaths(components) {
			pathWriters[p]++
		}
	}

	w := toggleableWalker{topology: topology, pathWriters: pathWriters}
	for _, key := range sortedKeys(sections) {
		section, _ := asMap(sections[key])
		components, _ := asMap(section["components"])
		w.walk(components, []string{key}, "")
	}
	return append(w.issues, unguardedCelIssues(topology, w.toggleables, w.celFields)...)
}

// toggleableWalker collects, for one topology, degraded-group issues, working
// toggleables and fields with CEL rules.
type toggleableWalker struct {
	topology    string
	pathWriters map[string]int

	issues      []Issue
	toggleables []toggleable
	celFields   []celField
}

// walk visits components under parent; parentToggleable is the key of the
// working toggleable they sit in, or "". Containers without a `components`
// map are skipped.
//
// Called with parent [advanced] and parentToggleable "" on:
//
//	monitoring:                  # toggleable
//	  uiType: group
//	  groupType: toggleable
//	  components:
//	    url:
//	      uiType: text
//	      path: spec.monitoring.url
//	      validation:
//	        celExpressions:
//	          - celExpr: "spec.monitoring.url != ''"
//	replicas:
//	  uiType: number
//	  path: spec.replicas
//	  validation:
//	    celExpressions:
//	      - celExpr: "spec.replicas > 0"
//
// records:
//
//	toggleables: [{key: "advanced.monitoring", paths: [spec.monitoring.url]}]
//	celFields:
//	  - {key: "advanced.monitoring.url", parentToggleable: "advanced.monitoring"}
//	  - {key: "advanced.replicas",       parentToggleable: ""}
func (w *toggleableWalker) walk(components map[string]any, parent []string, parentToggleable string) {
	for _, key := range sortedKeys(components) {
		item, _ := asMap(components[key])
		keyPath := slices.Concat(parent, []string{key})
		uiType, _ := item["uiType"].(string)
		if uiType != uiTypeGroup && uiType != uiTypeHidden {
			w.visitField(item, keyPath, parentToggleable)
			continue
		}
		children, ok := asMap(item["components"])
		if !ok {
			continue
		}
		childParent := parentToggleable
		if groupType, _ := item["groupType"].(string); uiType == uiTypeGroup && groupType == groupTypeToggleable {
			childParent = w.visitToggleable(children, keyPath, parentToggleable)
		}
		w.walk(children, keyPath, childParent)
	}
}

// visitField records a field that has CEL rules; fields without any are skipped.
//
// With keyPath [advanced replicas] and parentToggleable "":
//
//	uiType: number
//	path: spec.replicas
//	validation:
//	  celExpressions:
//	    - celExpr: "spec.replicas > 0"
//
// appends:
//
//	celField{key: "advanced.replicas", parentToggleable: "", exprs: ["spec.replicas > 0"]}
func (w *toggleableWalker) visitField(item map[string]any, keyPath []string, parentToggleable string) {
	exprs := celExpressions(item)
	if len(exprs) == 0 {
		return
	}
	w.celFields = append(w.celFields, celField{
		key:              strings.Join(keyPath, "."),
		parentToggleable: parentToggleable,
		exprs:            exprs,
	})
}

// visitToggleable records a toggleable group as working or degraded, and
// returns the parentToggleable for its children.
//
// With keyPath [advanced monitoring], parentToggleable "" and children:
//
//	url: {uiType: text, path: spec.monitoring.url}
//
// working: appends toggleable{key: "advanced.monitoring", paths: [spec.monitoring.url]}
// and returns "advanced.monitoring".
//
// degraded (e.g. another field also writes spec.monitoring.url): appends
// Issue{Rule: RuleToggleableOverlap, Group: "advanced.monitoring", ...} and
// returns "" (parentToggleable unchanged).
func (w *toggleableWalker) visitToggleable(children map[string]any, keyPath []string, parentToggleable string) string {
	paths := leafPaths(children)
	groupKey := strings.Join(keyPath, ".")
	if rule, msg := degradeRule(keyPath, paths, parentToggleable != "", w.pathWriters); rule != "" {
		w.issues = append(w.issues, Issue{
			Rule:     rule,
			Topology: w.topology,
			Group:    groupKey,
			Message:  msg + degradedSuffix,
		})
		return parentToggleable
	}
	w.toggleables = append(w.toggleables, toggleable{key: groupKey, paths: paths})
	return groupKey
}

// degradeRule returns the first rule that turns a toggleable group into a
// bordered one, with its message, or "" when the group works as a toggleable.
func degradeRule(keyPath, paths []string, nested bool, pathWriters map[string]int) (Rule, string) {
	switch {
	case slices.ContainsFunc(keyPath, isUnsafeKey):
		return RuleToggleableUnsafeKey, "has a section or group key outside [A-Za-z0-9_-], so it cannot name a switch"
	case len(paths) == 0:
		return RuleToggleableNoFields, "has no field with an API path, so there is nothing to switch off"
	case nested:
		return RuleToggleableNested, "is nested inside another toggleable group"
	}
	if overlap := firstSharedPath(paths, pathWriters); overlap != "" {
		return RuleToggleableOverlap, fmt.Sprintf("shares the path %q with a field outside it; switching it off would delete that field's value", overlap)
	}
	return "", ""
}

// unguardedCelIssues reports CEL rules that read a path of a working toggleable
// group without has(). While the group is off its paths are absent, so such a
// rule fails. Fields inside the group are skipped: they are off with it.
//
//	monitoring:                     # toggleable, paths [spec.monitoring.url]
//	  uiType: group
//	  groupType: toggleable
//	  components:
//	    url: {uiType: text, path: spec.monitoring.url}
//	replicas:
//	  uiType: number
//	  path: spec.replicas
//	  validation:
//	    celExpressions:
//	      - celExpr: "spec.monitoring.url != ''"                              # reported
//	      - celExpr: "!has(spec.monitoring.url) || spec.monitoring.url != ''" # ok
func unguardedCelIssues(topology string, toggleables []toggleable, celFields []celField) []Issue {
	var issues []Issue
	for _, field := range celFields {
		for _, group := range toggleables {
			if group.key == field.parentToggleable {
				continue
			}
			if path := firstUnguardedRef(field.exprs, group.paths); path != "" {
				issues = append(issues, Issue{
					Rule:     RuleToggleableCELUnguarded,
					Topology: topology,
					Group:    group.key,
					Field:    field.key,
					Message: fmt.Sprintf("a CEL rule reads %q of toggleable group %q without has(%s); it fails while the group is switched off",
						path, group.key, path,
					),
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
//
// With groupPaths [spec.monitoring.url]:
//
//	"spec.monitoring.url != ''"                              -> spec.monitoring.url
//	"spec.monitoring.url.size() > 0"                         -> spec.monitoring.url
//	"!has(spec.monitoring.url) || spec.monitoring.url != ''" -> ""
//	"original.spec.monitoring.url == ''"                     -> ""
func firstUnguardedRef(exprs, groupPaths []string) string {
	for _, expr := range exprs {
		guarded := make(map[string]bool)
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

// celExpressions returns the CEL expressions of a field in every form mode:
// `validation.celExpressions` and `validation.modes.<mode>.celExpressions`.
//
// item is a single field:
//
//	uiType: number
//	path: spec.replicas
//	validation:
//	  celExpressions:
//	    - celExpr: "spec.replicas > 0"
//	  modes:
//	    edit:
//	      celExpressions:
//	        - celExpr: "spec.replicas >= original.spec.replicas"
//
// returns:
//
//	["spec.replicas > 0", "spec.replicas >= original.spec.replicas"]
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

// celExprList returns the celExpr strings of a `validation` (or mode) block:
//
//	celExpressions:
//	  - celExpr: "spec.replicas > 0"
//	    message: ...
//	  - celExpr: "spec.replicas < 10"
//	    message: ...
//
// returns:
//
//	["spec.replicas > 0", "spec.replicas < 10"]
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
	own := make(map[string]int)
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

// leafPaths lists the paths written by every field under components, in
// the UI's terms: groups (and `hidden` containers) are walked, hidden fields
// and items without a uiType write nothing, and `path` is a string or a list
// of strings (multi-path).
//
//	url:                         # -> [spec.monitoring.url]
//	  uiType: text
//	  path: spec.monitoring.url
//	resources:                   # -> [spec.cpu, spec.memory]
//	  uiType: resources
//	  path: [spec.cpu, spec.memory]
//	nested:                      # group: walked recursively
//	  uiType: group
//	  components: {...}
//	label:                       # no uiType: nothing
//	  path: spec.label
func leafPaths(components map[string]any) []string {
	var paths []string
	for _, key := range sortedKeys(components) {
		item, _ := asMap(components[key])
		uiType, ok := item["uiType"]
		if !ok {
			continue
		}

		switch uiType {
		case uiTypeGroup, uiTypeHidden:
			if children, ok := asMap(item["components"]); ok {
				paths = append(paths, leafPaths(children)...)
			}
		default:
			paths = append(paths, targetPaths(item["path"])...)
		}
	}
	return paths
}

// targetPaths normalizes a field's `path`, dropping empty and duplicate entries:
//
//	path: spec.replicas              # -> [spec.replicas]
//	path: [spec.cpu, spec.memory]    # -> [spec.cpu, spec.memory]
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

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
