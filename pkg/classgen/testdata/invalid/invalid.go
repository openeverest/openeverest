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

// Package invalid holds types classgen refuses to render as parameters.
package invalid

// NestedCEL carries a rule below its root.
type NestedCEL struct {
	// +kubebuilder:validation:XValidation:rule="self.size() > 0",message="name must not be empty"
	Name string `json:"name"`
}

// RootCEL carries a rule on its root.
//
// +kubebuilder:validation:XValidation:rule="has(self.name)",message="name is required"
type RootCEL struct {
	// +optional
	Name string `json:"name,omitempty"`
}

// ClosedMap closes what is not a struct.
//
// +openeverest:validation:Closed
type ClosedMap map[string]string

// NotAnObject does not render to an object.
type NotAnObject string

// Tag is also declared in package params.
type Tag struct {
	Name string `json:"name"`
}
