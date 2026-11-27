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

const (
	uiTypeMap     = "map"
	uiTypeList    = "list"
	uiTypeText    = "text"
	groupTypeLine = "line"
)

// rowsKind is what tells a map from a list, the two fields the UI edits as
// rows. Everything else about them is checked the same way, as the UI does
// it (ui-generator utils/rows-field).
type rowsKind struct {
	// uiType names the field in a message.
	uiType string
	rules  rowsRules
	// isSupported reports whether an item template is the one the UI
	// implements, which supportedItemTemplate describes.
	isSupported           func(itemTemplate any) bool
	supportedItemTemplate string
	// rowFields returns the fields of a row as the schema writes them.
	rowFields func(itemTemplate any) []rowField
	// ignoredRules and ignoredRowFieldRules are the rules a schema may set
	// that do nothing here, besides the ones no rows field acts on (`regex`
	// on the field, CEL on a row's field): on the field itself, and on a
	// field of its row.
	ignoredRules         []string
	ignoredRowFieldRules []string
	// isValidDefault reports whether a default is in the shape the field
	// writes, which valueShape names.
	isValidDefault func(value any) bool
	valueShape     string
	// countEntries returns the number of entries a valid default holds.
	countEntries func(defaultValue any) int
}

// rowsRules are the rules of one kind, named after its uiType.
type rowsRules struct {
	unsupported         Rule
	multiPath           Rule
	overlap             Rule
	ignoredOption       Rule
	invalidDefault      Rule
	invalidLimits       Rule
	unsatisfiableLimits Rule
}

// rowField is a field of a row with where the schema writes it:
// "itemTemplate.components.key" in a map, "itemTemplate" in a list.
type rowField struct {
	at   string
	item map[string]any
}

// rowsKindOf returns the kind of field a uiType names, and false for a field
// the UI doesn't edit as rows.
func rowsKindOf(uiType any) (rowsKind, bool) {
	switch uiType {
	case uiTypeMap:
		return rowsKind{
			uiType: uiTypeMap,
			rules: rowsRules{
				unsupported:         RuleMapUnsupported,
				multiPath:           RuleMapMultiPath,
				overlap:             RuleMapOverlap,
				ignoredOption:       RuleMapIgnoredOption,
				invalidDefault:      RuleMapInvalidDefault,
				invalidLimits:       RuleMapInvalidLimits,
				unsatisfiableLimits: RuleMapUnsatisfiableLimits,
			},
			isSupported:           isKeyValueRow,
			supportedItemTemplate: `a group, with groupType "line" or none, of exactly a text field with path "key" and a text field with path "value", which is the only row supported yet`,
			rowFields:             mapRowFields,
			// A map holds a key once whatever `unique` says.
			ignoredRules:   []string{"unique"},
			isValidDefault: isStringMap,
			valueShape:     "map of strings",
			countEntries:   countMapEntries,
		}, true
	case uiTypeList:
		return rowsKind{
			uiType: uiTypeList,
			rules: rowsRules{
				unsupported:         RuleListUnsupported,
				multiPath:           RuleListMultiPath,
				overlap:             RuleListOverlap,
				ignoredOption:       RuleListIgnoredOption,
				invalidDefault:      RuleListInvalidDefault,
				invalidLimits:       RuleListInvalidLimits,
				unsatisfiableLimits: RuleListUnsatisfiableLimits,
			},
			isSupported:           isTextItem,
			supportedItemTemplate: "a text field with no path or id, which is the only item supported yet",
			rowFields:             listRowFields,
			// A row is an entry only once it holds a value, so none is ever
			// missing.
			ignoredRowFieldRules: []string{"required"},
			isValidDefault:       isStringList,
			valueShape:           "list of strings",
			countEntries:         countListEntries,
		}, true
	}
	return rowsKind{}, false
}

// mapRowRoles returns the paths a map row's two fields write in the row.
func mapRowRoles() []string {
	return []string{"key", "value"}
}

// isKeyValueRow reports whether an item template is the one row of a map the
// UI implements: a group, in a line or stacked, of exactly a text field at
// `key` and a text field at `value`. The fields are told apart by path, so
// their names are the schema's own:
//
//	itemTemplate:
//	  uiType: group
//	  groupType: line          # or none: the fields are stacked
//	  components:
//	    name: {uiType: text, path: key}
//	    setting: {uiType: text, path: value}
func isKeyValueRow(itemTemplate any) bool {
	row, _ := itemTemplate.(map[string]any)
	if row["uiType"] != uiTypeGroup {
		return false
	}
	if groupType, ok := row["groupType"]; ok && groupType != groupTypeLine {
		return false
	}
	components, _ := row["components"].(map[string]any)
	if len(components) != len(mapRowRoles()) {
		return false
	}
	for _, role := range mapRowRoles() {
		if rowFieldName(components, role) == "" {
			return false
		}
	}
	return true
}

// rowFieldName returns the name of the row's text field writing role, or "".
func rowFieldName(components map[string]any, role string) string {
	for _, name := range sortedKeys(components) {
		component, _ := components[name].(map[string]any)
		if _, isGroup := component["components"]; isGroup {
			continue
		}
		if component["uiType"] == uiTypeText && component["path"] == role {
			return name
		}
	}
	return ""
}

// mapRowFields returns the fields of the group a map declares as its row.
func mapRowFields(itemTemplate any) []rowField {
	row, _ := itemTemplate.(map[string]any)
	components, _ := row["components"].(map[string]any)
	fields := make([]rowField, 0, len(components))
	for _, name := range sortedKeys(components) {
		item, _ := components[name].(map[string]any)
		fields = append(fields, rowField{at: "itemTemplate.components." + name, item: item})
	}
	return fields
}

// isTextItem reports whether an item template is the one entry of a list the
// UI implements: a text field written directly, with no `path` or `id`, since
// the field is the entry:
//
//	itemTemplate:
//	  uiType: text
//	  fieldParams: {placeholder: 10.0.0.0/8}
func isTextItem(itemTemplate any) bool {
	item, _ := itemTemplate.(map[string]any)
	if item["uiType"] != uiTypeText {
		return false
	}
	for _, key := range []string{"components", "path", "id"} {
		if _, ok := item[key]; ok {
			return false
		}
	}
	return true
}

// listRowFields returns the one field a list declares, which is its row.
func listRowFields(itemTemplate any) []rowField {
	item, _ := itemTemplate.(map[string]any)
	return []rowField{{at: "itemTemplate", item: item}}
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

func isStringList(value any) bool {
	list, ok := value.([]any)
	if !ok {
		return false
	}
	for _, entry := range list {
		if _, ok := entry.(string); !ok {
			return false
		}
	}
	return true
}

// countMapEntries returns the entries the form reads from a default map: an
// entry with neither a key nor a value is an empty row, which is no entry.
func countMapEntries(defaultValue any) int {
	entries, _ := defaultValue.(map[string]any)
	count := 0
	for key, value := range entries {
		if key != "" || value != "" {
			count++
		}
	}
	return count
}

// countListEntries returns the entries the form reads from a default list:
// an empty string is an empty row, which is no entry.
func countListEntries(defaultValue any) int {
	entries, _ := defaultValue.([]any)
	count := 0
	for _, entry := range entries {
		if entry != "" {
			count++
		}
	}
	return count
}
