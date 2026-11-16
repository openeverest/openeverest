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

// Package params holds parameters types for the classgen tests.
package params

import "k8s.io/apimachinery/pkg/api/resource"

// Mode is how the agent connects.
//
// +kubebuilder:validation:Enum=plain;tls
type Mode string

// Parameters exercise the markers a parameters type carries.
type Parameters struct {
	// URL of the server.
	// +kubebuilder:validation:Pattern=`^https?://`
	// +kubebuilder:validation:MaxLength=2048
	URL string `json:"url"`

	// Mode of the connection.
	// +kubebuilder:default=tls
	// +optional
	Mode Mode `json:"mode,omitempty"`

	// Retries bounds the connection attempts.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +optional
	Retries *int32 `json:"retries,omitempty"`

	// Resources of the agent.
	// +optional
	Resources *Resources `json:"resources,omitempty"`

	// Tags label the server.
	// +kubebuilder:validation:MaxItems=4
	// +optional
	Tags []Tag `json:"tags,omitempty"`
}

// Resources are the requests and limits of the agent.
//
// +openeverest:validation:Closed
type Resources struct {
	// Limits of the agent.
	// +optional
	Limits map[string]resource.Quantity `json:"limits,omitempty"`
	// Requests of the agent.
	// +optional
	Requests map[string]resource.Quantity `json:"requests,omitempty"`
}

// Tag is a label.
type Tag struct {
	// Name of the tag.
	// +kubebuilder:validation:Pattern=`^[a-z]+$`
	Name string `json:"name"`
}
