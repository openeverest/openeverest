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

package preset

// parametersRef references a parameters entry. The value may be a plain string or a
// ref-like object such as {"name": ""}; both shapes are handled transparently.
// Writes flip the shared dirty flag so the traversal knows to re-marshal.
type parametersRef struct {
	meta

	parent map[string]any
	key    string
	dirty  *bool
}

func (r parametersRef) IsEmpty() bool {
	switch value := r.parent[r.key].(type) {
	case string:
		return value == ""
	case map[string]any:
		name, _ := value["name"].(string)
		namespace, _ := value["namespace"].(string)

		return name == "" && namespace == ""
	}
	return false
}

func (r parametersRef) Set(namespace, name string) {
	if obj, ok := r.parent[r.key].(map[string]any); ok {
		obj["name"] = name

		if _, ok := obj["namespace"]; ok {
			obj["namespace"] = namespace
		}
	} else {
		r.parent[r.key] = name
	}
	*r.dirty = true
}
