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
	"slices"
)

// Rule identifies a schema rule; values are stable for tooling and the shared
// test cases.
type Rule string

// Rules checked by Validate, grouped by validator; see README.md for their
// exact conditions.
const (
	// validateToggleables
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

// topologyValidator checks the sections of one topology.
type topologyValidator func(topology string, sections map[string]any) []Issue

// topologyValidators run on every topology; register new validators here.
var topologyValidators = []topologyValidator{
	validateToggleables,
}

// Validate checks a provider UI schema. Issues are sorted by topology, group
// and field.
func Validate(in Input) []Issue {
	var issues []Issue
	for _, topology := range sortedKeys(in.UISchema) {
		sections, ok := topologySections(in.UISchema[topology])
		if !ok {
			continue
		}
		for _, validate := range topologyValidators {
			issues = append(issues, validate(topology, sections)...)
		}
	}
	slices.SortStableFunc(issues, compareIssues)
	return issues
}

func topologySections(topology any) (map[string]any, bool) {
	schema, _ := asMap(topology)
	return asMap(schema["sections"])
}

func compareIssues(a, b Issue) int {
	return cmp.Or(
		cmp.Compare(a.Topology, b.Topology),
		cmp.Compare(a.Group, b.Group),
		cmp.Compare(a.Field, b.Field),
	)
}
