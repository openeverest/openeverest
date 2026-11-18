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
	"strconv"
	"strings"
)

const (
	uiTypeSelect         = "select"
	ignoredDefaultSuffix = "; the UI ignores it and pre-selects the Provider default or the first option"
)

// validateSelectDefaults mirrors how preprocessSchema in the UI resolves a
// select that reads its options from the Provider: a default that is not one
// of those options is never pre-selected.
//
// sections is one topology's `sections` map:
//
//	base:                                    # section key
//	  components:
//	    version:                             # Field "base.version"
//	      uiType: select
//	      path: spec.version
//	      fieldParams:
//	        optionsPath: spec.versions       # options: the Provider's spec.versions[*].name
//	        optionsPathConfig:
//	          labelPath: name
//	          valuePath: name
//	          defaultValuePath: spec.defaultVersion   # must name one of the options
//	        defaultValue: "8.0"                       # must be one of the options
func validateSelectDefaults(topology string, sections, provider map[string]any) []Issue {
	w := selectDefaultWalker{topology: topology, provider: provider}
	for _, key := range sortedKeys(sections) {
		section, _ := sections[key].(map[string]any)
		components, _ := section["components"].(map[string]any)
		w.walk(components, []string{key})
	}
	return w.issues
}

// selectDefaultWalker collects, for one topology, the select defaults that are
// not among the options built from the Provider.
type selectDefaultWalker struct {
	topology string
	provider map[string]any

	issues []Issue
}

// walk checks every select under components; containers are walked the same
// way as in getPaths.
func (w *selectDefaultWalker) walk(components map[string]any, parent []string) {
	for _, key := range sortedKeys(components) {
		item, _ := components[key].(map[string]any)
		currentPath := slices.Concat(parent, []string{key})
		switch item["uiType"] {
		case uiTypeGroup, uiTypeHidden:
			if children, ok := item["components"].(map[string]any); ok {
				w.walk(children, currentPath)
			}
		case uiTypeSelect:
			w.check(item, strings.Join(currentPath, "."))
		}
	}
}

// check reports the defaults of one select that are not among its options. A
// select with static `options`, or whose `optionsPath` resolves to no options,
// is not checked.
func (w *selectDefaultWalker) check(item map[string]any, field string) {
	fieldParams, _ := item["fieldParams"].(map[string]any)
	optionsPath, _ := fieldParams["optionsPath"].(string)
	config, _ := fieldParams["optionsPathConfig"].(map[string]any)
	if optionsPath == "" || config == nil {
		return
	}
	valuePath, _ := config["valuePath"].(string)
	options := selectOptionValues(w.provider, optionsPath, valuePath)
	if len(options) == 0 {
		return
	}

	if problem := invalidSchemaDefault(fieldParams["defaultValue"], options); problem != "" {
		w.issues = append(w.issues, Issue{
			Rule:     RuleSelectDefaultNotAnOption,
			Topology: w.topology,
			Field:    field,
			Message:  fmt.Sprintf("%s built from %q%s", problem, optionsPath, ignoredDefaultSuffix),
		})
	}

	defaultValuePath, _ := config["defaultValuePath"].(string)
	if value := stringify(getByPath(w.provider, defaultValuePath)); value != "" && !slices.Contains(options, value) {
		w.issues = append(w.issues, Issue{
			Rule:     RuleSelectProviderDefaultNotAnOption,
			Topology: w.topology,
			Field:    field,
			Message: fmt.Sprintf("the Provider default %q at %q is not one of the options built from %q with valuePath %q; the UI pre-selects the first option instead",
				value, defaultValuePath, optionsPath, valuePath,
			),
		})
	}
}

// invalidSchemaDefault returns why the UI cannot pre-select a schema
// `defaultValue`, or "" when it can.
//
// With options [8.0, 8.4]:
//
//	defaultValue: "8.4"   -> ""
//	defaultValue: ""      -> ""   (an explicitly empty selection)
//	defaultValue: "9.9"   -> not one of the options
//	defaultValue: 8.4     -> not a string: YAML reads it as a number
func invalidSchemaDefault(value any, options []string) string {
	switch v := value.(type) {
	case nil, []any:
		// Unset or null, or the list default of a `multiple` select.
		return ""
	case string:
		if v == "" || slices.Contains(options, v) {
			return ""
		}
		return fmt.Sprintf("defaultValue %q is not one of the options", v)
	default:
		return fmt.Sprintf("defaultValue %v is not a string (quote it), so it equals none of the options", v)
	}
}

// selectOptionValues returns the option values the UI builds for a select:
// one per entry of the list at optionsPath, read at valuePath.
//
// With optionsPath spec.versions and valuePath name:
//
//	spec:
//	  versions:
//	    - name: "8.0"     # -> "8.0"
//	    - name: "8.4"     # -> "8.4"
//	    - {}              # -> ""
func selectOptionValues(provider map[string]any, optionsPath, valuePath string) []string {
	list, ok := getByPath(provider, optionsPath).([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(list))
	for _, entry := range list {
		item, _ := entry.(map[string]any)
		values = append(values, stringify(getByPath(item, valuePath)))
	}
	return values
}

// getByPath returns the value at a dot-separated path, or nil when the path is
// empty or leaves the nested maps; lists are not indexed.
func getByPath(obj map[string]any, path string) any {
	if path == "" {
		return nil
	}
	var current any = obj
	for key := range strings.SplitSeq(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

// stringify mirrors `String(value ?? "")` in the UI for the scalars a Provider holds.
func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}
