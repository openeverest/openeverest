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
// The UI applies the same rules at runtime (ui-generator utils/map-field and
// utils/toggleable, both run by preprocessSchema); testdata/*.yaml is the
// shared contract both implementations are tested against (see
// preprocess-shared-cases.test.ts in the UI), so keep them in sync.
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
	RuleMapUnsupported         Rule = "map-unsupported"
	RuleMapMultiPath           Rule = "map-multi-path"
	RuleMapOverlap             Rule = "map-overlap"
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
	// by group keys, e.g. "advanced.monitoring". Empty for map rules.
	Group string
	// Field is the dotted key path of the field a map rule is about, or of
	// the field whose CEL rule reads the group; empty for rules about the group
	// itself.
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
//	      components:        # walked by validateMaps and validateToggleables
//	        monitoring: {...}
//	sharded:
//	  sections: {...}
func Validate(in Input) []Issue {
	var issues []Issue
	for _, topology := range sortedKeys(in.UISchema) {
		topologySchema, _ := in.UISchema[topology].(map[string]any)
		sections, ok := topologySchema["sections"].(map[string]any)
		if !ok {
			continue
		}
		// The UI drops these map fields before resolving toggleables, so the
		// toggleable rules see them as writing nothing.
		mapIssues, droppedMaps := validateMaps(topology, sections)
		issues = append(issues, mapIssues...)
		issues = append(issues, validateToggleables(topology, sections, droppedMaps)...)
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

const (
	uiTypeMap               = "map"
	uiTypeWidget            = "widget"
	widgetTypeKeyValue      = "keyValue"
	droppedMapSuffix        = "; the UI does not render it"
	mapOwnsValueSentence    = "a map field must own its value alone"
	mapReadsOnePathSentence = "an edit reads only the first one, so the others would drift from it"
)

// field is a field found by collectFields: its dotted key path, the item and
// the API paths it writes.
type field struct {
	key   string
	item  map[string]any
	paths []string
}

// validateMaps mirrors resolveMapFields in the UI: a map field owns the whole
// value at its path, so the UI renders only a shape it implements, with one
// path, and only when no other field writes that path, a path inside it or
// one it sits in. It returns the issues and the key paths of the map fields
// the UI drops. A dropped field writes nothing, so it never makes another map
// overlap.
//
// sections is one topology's `sections` map:
//
//	advanced:
//	  components:
//	    labels:                  # rendered: a key/value map owning spec.labels
//	      uiType: map
//	      path: spec.labels
//	      itemTemplate: {uiType: widget, widgetType: keyValue}
//	    users:                   # map-unsupported: rows from a custom group
//	      uiType: map
//	      path: spec.users
//	      itemTemplate: {uiType: group, components: {...}}
func validateMaps(topology string, sections map[string]any) ([]Issue, map[string]bool) {
	var issues []Issue
	dropped := make(map[string]bool)
	var writers []field
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		for _, f := range collectFields(components, []string{key}) {
			if rule, msg := mapDropRule(f); rule != "" {
				dropped[f.key] = true
				issues = append(issues, Issue{
					Rule:     rule,
					Topology: topology,
					Field:    f.key,
					Message:  msg + droppedMapSuffix,
				})
				continue
			}
			writers = append(writers, f)
		}
	}

	for _, m := range writers {
		if m.item["uiType"] != uiTypeMap {
			continue
		}
		if other, path := findOverlap(m, writers); other != "" {
			dropped[m.key] = true
			issues = append(issues, Issue{
				Rule:     RuleMapOverlap,
				Topology: topology,
				Field:    m.key,
				Message: fmt.Sprintf("overlaps the path %q of field %q, while %s%s",
					path, other, mapOwnsValueSentence, droppedMapSuffix,
				),
			})
		}
	}
	return issues, dropped
}

// mapDropRule returns the rule a map field breaks on its own, with its
// message, or "" when it has none (or is not a map). The order matches
// resolveMapFields in the UI.
func mapDropRule(f field) (Rule, string) {
	if f.item["uiType"] != uiTypeMap {
		return "", ""
	}
	if !isKeyValueTemplate(f.item) {
		return RuleMapUnsupported, describeUnsupportedMap(f.item)
	}
	if len(f.paths) > 1 {
		return RuleMapMultiPath, fmt.Sprintf("has %d paths, while %s", len(f.paths), mapReadsOnePathSentence)
	}
	return "", ""
}

// isKeyValueTemplate reports whether a map field's rows come from the one
// item template the UI implements:
//
//	itemTemplate: {uiType: widget, widgetType: keyValue}
func isKeyValueTemplate(item map[string]any) bool {
	itemTemplate, _ := item["itemTemplate"].(map[string]any)
	return itemTemplate["uiType"] == uiTypeWidget && itemTemplate["widgetType"] == widgetTypeKeyValue
}

func describeUnsupportedMap(item map[string]any) string {
	itemTemplate, _ := item["itemTemplate"].(map[string]any)
	uiType, _ := itemTemplate["uiType"].(string)
	widgetType, _ := itemTemplate["widgetType"].(string)
	return fmt.Sprintf("has an itemTemplate with uiType %q and widgetType %q, while only uiType %q with widgetType %q is supported",
		uiType, widgetType, uiTypeWidget, widgetTypeKeyValue,
	)
}

// findOverlap returns the first other field writing a path that overlaps one
// of the map's paths, with that path, or "" when there is none.
func findOverlap(m field, writers []field) (string, string) {
	for _, other := range writers {
		if other.key == m.key {
			continue
		}
		for _, path := range other.paths {
			if slices.ContainsFunc(m.paths, func(own string) bool { return pathsOverlap(path, own) }) {
				return other.key, path
			}
		}
	}
	return "", ""
}

// pathsOverlap reports whether two paths are the same or one sits inside the
// other: spec.labels overlaps spec.labels and spec.labels.team, but not
// spec.labelsMode.
func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+".") || strings.HasPrefix(b, a+".")
}

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
func validateToggleables(topology string, sections map[string]any, droppedMaps map[string]bool) []Issue {
	// pathWriters counts the fields writing each path across the topology.
	pathWriters := make(map[string]int)
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		for _, p := range getPaths(components, []string{key}, droppedMaps) {
			pathWriters[p]++
		}
	}

	w := toggleableWalker{topology: topology, pathWriters: pathWriters, droppedMaps: droppedMaps}
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		w.walk(components, []string{key}, "")
	}
	return append(w.issues, reportCelWithoutGuard(topology, w.toggleables, w.celFields)...)
}

// toggleableWalker collects, for one topology, degraded-group issues, working
// toggleables and fields with CEL rules.
type toggleableWalker struct {
	topology    string
	pathWriters map[string]int
	droppedMaps map[string]bool

	issues      []Issue
	toggleables []toggleable
	celFields   []celField
}

// walk records toggleable and cel fields recursively for components under parent;
// parentToggleable is the key of the working toggleable they sit in, or "".
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
		current, _ := components[key].(map[string]any)
		currentPath := slices.Concat(parent, []string{key})
		uiType, _ := current["uiType"].(string)
		if uiType != uiTypeGroup && uiType != uiTypeHidden {
			// A dropped map field isn't rendered, so its rules never run.
			if !w.droppedMaps[strings.Join(currentPath, ".")] {
				w.recordCelExpressions(current, currentPath, parentToggleable)
			}
			continue
		}
		children, ok := current["components"].(map[string]any)
		if !ok {
			continue
		}

		childrenToggleable := parentToggleable
		if groupType, _ := current["groupType"].(string); uiType == uiTypeGroup && groupType == groupTypeToggleable {
			if w.recordToggleable(children, currentPath, parentToggleable != "") {
				childrenToggleable = strings.Join(currentPath, ".")
			}
		}
		w.walk(children, currentPath, childrenToggleable)
	}
}

// recordCelExpressions records a field that has CEL rules; fields without any are skipped.
func (w *toggleableWalker) recordCelExpressions(item map[string]any, currentPath []string, parentToggleable string) {
	exprs := getCelExpressions(item)
	if len(exprs) == 0 {
		return
	}
	w.celFields = append(w.celFields, celField{
		key:              strings.Join(currentPath, "."),
		parentToggleable: parentToggleable,
		exprs:            exprs,
	})
}

// recordToggleable records a toggleable group and reports whether it works as
// one. A working group is added to toggleables; a degraded one (see
// degradeRule) is reported as an issue instead. isToggleable is whether the group
// sits inside a working toggleable.
func (w *toggleableWalker) recordToggleable(children map[string]any, currentPath []string, isToggleable bool) bool {
	paths := getPaths(children, currentPath, w.droppedMaps)
	groupKey := strings.Join(currentPath, ".")
	if rule, msg := degradeRule(currentPath, paths, isToggleable, w.pathWriters); rule != "" {
		w.issues = append(w.issues, Issue{
			Rule:     rule,
			Topology: w.topology,
			Group:    groupKey,
			Message:  msg + degradedSuffix,
		})
		return false
	}
	w.toggleables = append(w.toggleables, toggleable{key: groupKey, paths: paths})
	return true
}

// degradeRule returns the first rule that makes a toggleable group into
// degraded mode with its message. Returns "" when the group is working.
// The check order matches resolveToggleable in the UI and decides which rule
// is reported when several apply; keep them in sync.
func degradeRule(keyPath, paths []string, nested bool, pathWriters map[string]int) (Rule, string) {
	switch {
	case slices.ContainsFunc(keyPath, isUnsafeKey):
		return RuleToggleableUnsafeKey, "has a section or group key outside [A-Za-z0-9_-], so it cannot name a switch"
	case len(paths) == 0:
		return RuleToggleableNoFields, "has no field with an API path, so there is nothing to switch off"
	case nested:
		return RuleToggleableNested, "is nested inside another toggleable group"
	}
	if overlap := pathWrittenOutside(paths, pathWriters); overlap != "" {
		return RuleToggleableOverlap, fmt.Sprintf("shares the path %q with a field outside it; switching it off would delete that field's value", overlap)
	}
	return "", ""
}

// reportCelWithoutGuard reports CEL rules that read a path of a toggleable
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
func reportCelWithoutGuard(topology string, toggleables []toggleable, celFields []celField) []Issue {
	var issues []Issue
	for _, field := range celFields {
		for _, group := range toggleables {
			if group.key == field.parentToggleable {
				continue
			}
			if path := getCelWithoutGuard(field.exprs, group.paths); path != "" {
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

// getCelWithoutGuard returns a path from groupPaths without a has(), or "".
//
// With groupPaths [spec.monitoring.url]:
//
//	"spec.monitoring.url != ''"                              -> spec.monitoring.url
//	"spec.monitoring.url.size() > 0"                         -> spec.monitoring.url
//	"!has(spec.monitoring.url) || spec.monitoring.url != ''" -> ""
//	"original.spec.monitoring.url == ''"                     -> ""
func getCelWithoutGuard(exprs, groupPaths []string) string {
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

// getCelExpressions returns the CEL expressions of a field in every form mode:
// `validation.getCelExpressions` and `validation.modes.<mode>.getCelExpressions`.
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
func getCelExpressions(item map[string]any) []string {
	validation, _ := item["validation"].(map[string]any)
	exprs := getCelExprs(validation)
	modes, _ := validation["modes"].(map[string]any)
	for _, mode := range sortedKeys(modes) {
		branch, _ := modes[mode].(map[string]any)
		exprs = append(exprs, getCelExprs(branch)...)
	}
	return exprs
}

// getCelExprs returns the celExpr strings of a `validation` (or mode) block:
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
func getCelExprs(validation map[string]any) []string {
	list, _ := validation["celExpressions"].([]any)
	var exprs []string
	for _, entry := range list {
		m, _ := entry.(map[string]any)
		if expr, ok := m["celExpr"].(string); ok && expr != "" {
			exprs = append(exprs, expr)
		}
	}
	return exprs
}

// pathWrittenOutside returns the first path in groupPaths that a field
// outside the group also writes, or "".
func pathWrittenOutside(groupPaths []string, topologyPaths map[string]int) string {
	groupPathsMap := make(map[string]int)
	for _, p := range groupPaths {
		groupPathsMap[p]++
	}
	for _, p := range groupPaths {
		if topologyPaths[p] > groupPathsMap[p] {
			return p
		}
	}
	return ""
}

// getPaths returns the paths written by the fields under components (see
// collectFields), except the map fields in droppedMaps; parent is the key
// path of components.
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
func getPaths(components map[string]any, parent []string, droppedMaps map[string]bool) []string {
	var paths []string
	for _, f := range collectFields(components, parent) {
		if !droppedMaps[f.key] {
			paths = append(paths, f.paths...)
		}
	}
	return paths
}

// collectFields returns the fields under components in key order, depth
// first, keyed under parent. The uiType `group` and `hidden` are containers,
// searched recursively when they contain components; an item without uiType
// is not a field.
func collectFields(components map[string]any, parent []string) []field {
	var fields []field
	for _, key := range sortedKeys(components) {
		item, _ := components[key].(map[string]any)
		uiType, ok := item["uiType"]
		if !ok {
			continue
		}

		keyPath := slices.Concat(parent, []string{key})
		switch uiType {
		case uiTypeGroup, uiTypeHidden:
			if children, ok := item["components"].(map[string]any); ok {
				fields = append(fields, collectFields(children, keyPath)...)
			}
		default:
			fields = append(fields, field{
				key:   strings.Join(keyPath, "."),
				item:  item,
				paths: normalizePaths(item["path"]),
			})
		}
	}
	return fields
}

// normalizePaths normalizes a field's `path` based on its type,
// and drops empty and duplicate entries:
//
//	path: spec.replicas              # -> [spec.replicas]
//	path: [spec.cpu, spec.memory]    # -> [spec.cpu, spec.memory]
func normalizePaths(path any) []string {
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

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
