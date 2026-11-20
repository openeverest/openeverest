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

// Package view holds a view type for the classgen tests.
package view

import "github.com/openeverest/openeverest/v2/pkg/classgen/testdata/params"

// StorageView is the spec of the objects that reference a class.
//
// +kubebuilder:validation:XValidation:rule="!self.location.bucket.contains('..')",message="bucket must not contain ..",fieldPath=".location.bucket"
// +kubebuilder:validation:XValidation:rule="!has(self.tls) || self.parameters.url.startsWith('https://')",message="tls needs an https url",fieldPath=".tls"
type StorageView struct {
	Location   Location          `json:"location"`
	Parameters params.Parameters `json:"parameters"`
	// +optional
	TLS *TLS `json:"tls,omitempty"`
}

// Location is where the data goes.
type Location struct {
	Bucket string `json:"bucket"`
}

// TLS is how the server is verified.
type TLS struct {
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}
