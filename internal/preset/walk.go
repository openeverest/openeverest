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

import (
	"encoding/json"
	"strings"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// WalkSpec visits every resolvable resource reference in the spec. Structured
// fields (Backup.ClassRef, Backup.Storages[].StorageRef, Storage.StorageClass)
// and Parameters entries are all surfaced as FieldRef values. Mutations performed
// by the visitor via FieldRef.Set are written back into the spec, including
// re-marshalling any Parameters whose contents changed.
func WalkSpec(spec *corev1alpha1.InstanceSpec, visit func(FieldRef) error) error {
	if spec == nil {
		return nil
	}
	for name := range spec.Components {
		component := spec.Components[name]
		if err := walkComponent(name, &component, visit); err != nil {
			return err
		}
		spec.Components[name] = component
	}
	return nil
}

// walkComponent surfaces the typed struct references and the customSpec entries
// of a single component.
func walkComponent(name string, component *corev1alpha1.ComponentSpec, visit func(FieldRef) error) error {
	if component.Storage != nil {
		ref := storageClassRef{meta: newMeta(name, KindStorageClass, "storage.storageClass"), storage: component.Storage}
		if err := visit(ref); err != nil {
			return err
		}
	}

	if component.Parameters == nil || len(component.Parameters.Raw) == 0 {
		return nil
	}

	var data map[string]any
	if err := json.Unmarshal(component.Parameters.Raw, &data); err != nil {
		return err
	}

	var dirty bool
	if err := walkParameters(name, data, "parameters", &dirty, visit); err != nil {
		return err
	}

	if dirty {
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		component.Parameters.Raw = raw
	}

	return nil
}

// walkParameters recursively surfaces known reference fields within a Parameters
// object. A key that matches a registered alias is treated as a reference (even
// when its value is an object); any other object value is descended into.
func walkParameters(component string, data map[string]any, path string, dirty *bool, visit func(FieldRef) error) error {
	for key, value := range data {
		path = strings.Join([]string{path, key}, ".")
		if rk, ok := aliasKind(key); ok {
			ref := parametersRef{
				meta:   meta{component: component, kind: rk.kind, scope: rk.scope, path: path},
				parent: data,
				key:    key,
				dirty:  dirty,
			}
			if err := visit(ref); err != nil {
				return err
			}
			continue
		}

		if nested, ok := value.(map[string]any); ok {
			if err := walkParameters(component, nested, path, dirty, visit); err != nil {
				return err
			}
		}
	}
	return nil
}
