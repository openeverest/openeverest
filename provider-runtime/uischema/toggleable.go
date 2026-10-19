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
	"regexp"
	"slices"
	"strings"
)

const (
	groupTypeToggleable = "toggleable"
	degradedSuffix      = "; the UI renders it as a bordered group"
)

var (
	// Keys name the group's form-only switch; the UI rejects anything else.
	switchKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

	// A `spec.` path not preceded by an identifier or a dot, so `original.spec.x` is skipped.
	celSpecRef = regexp.MustCompile(`(^|[^\w.])(spec(?:\.\w+)+)`)
	celHasCall = regexp.MustCompile(`has\(\s*(spec(?:\.\w+)+)\s*\)`)
)

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

// toggleableScan collects what validateToggleables needs from one walk over a topology.
type toggleableScan struct {
	topology  string
	pathUsage map[string]int
	issues    []Issue
	active    []toggleable
	celFields []celField
}

// validateToggleables mirrors resolveToggleable in the UI: switching a
// toggleable group off deletes every API path its fields write, so it must own
// those paths exclusively and must not sit inside another toggleable. CEL rules
// outside the group then see those paths as absent and must guard them.
func validateToggleables(topology string, sections map[string]any) []Issue {
	s := &toggleableScan{topology: topology, pathUsage: topologyPathUsage(sections)}
	forEachSection(sections, func(key string, components map[string]any) {
		s.walk(components, []string{key}, "")
	})
	return append(s.issues, s.unguardedCELIssues()...)
}

// walk visits components; enclosing is the key of the active toggleable they
// sit in, or "".
func (s *toggleableScan) walk(components map[string]any, parent []string, enclosing string) {
	for _, key := range sortedKeys(components) {
		item, _ := asMap(components[key])
		keyPath := slices.Concat(parent, []string{key})

		if !isContainer(item) {
			if exprs := celExpressions(item); len(exprs) > 0 {
				s.celFields = append(s.celFields, celField{key: strings.Join(keyPath, "."), enclosing: enclosing, exprs: exprs})
			}
			continue
		}

		children, ok := asMap(item["components"])
		if !ok {
			continue
		}
		childEnclosing := enclosing
		if isToggleableGroup(item) {
			if groupKey, ok := s.addToggleable(keyPath, children, enclosing != ""); ok {
				childEnclosing = groupKey
			}
		}
		s.walk(children, keyPath, childEnclosing)
	}
}

// addToggleable records the group as active, or reports why it degrades to a
// bordered group. It returns the group key and whether the group is active.
func (s *toggleableScan) addToggleable(keyPath []string, children map[string]any, nested bool) (string, bool) {
	groupKey := strings.Join(keyPath, ".")
	paths := leafPaths(children)
	if rule, msg := degradeRule(keyPath, paths, nested, s.pathUsage); rule != "" {
		s.issues = append(s.issues, Issue{Rule: rule, Topology: s.topology, Group: groupKey, Message: msg + degradedSuffix})
		return groupKey, false
	}
	s.active = append(s.active, toggleable{key: groupKey, paths: paths})
	return groupKey, true
}

func isToggleableGroup(item map[string]any) bool {
	uiType, _ := item["uiType"].(string)
	groupType, _ := item["groupType"].(string)
	return uiType == uiTypeGroup && groupType == groupTypeToggleable
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

func (s *toggleableScan) unguardedCELIssues() []Issue {
	var issues []Issue
	for _, field := range s.celFields {
		for _, group := range s.active {
			if group.key == field.enclosing {
				continue
			}
			path := firstUnguardedRef(field.exprs, group.paths)
			if path == "" {
				continue
			}
			issues = append(issues, Issue{
				Rule:     RuleToggleableCELUnguarded,
				Topology: s.topology,
				Group:    group.key,
				Field:    field.key,
				Message: fmt.Sprintf("a CEL rule reads %q of toggleable group %q without has(%s); it fails while the group is switched off",
					path, group.key, path,
				),
			})
		}
	}
	return issues
}

// firstUnguardedRef returns a group path that some expression reads (directly
// or below it) without a has() on that exact path, or "".
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
