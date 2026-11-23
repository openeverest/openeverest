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

package cli

import (
	"encoding/json"
	"io"

	"sigs.k8s.io/yaml"
)

// Encode writes v to w as YAML when asYAML is true, otherwise as JSON.
// YAML encoding follows json struct tags (via sigs.k8s.io/yaml).
func Encode(w io.Writer, v any, asYAML bool) error {
	if asYAML {
		b, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	}
	return json.NewEncoder(w).Encode(v)
}
