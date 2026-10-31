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

// Package schema wraps the parameter types of the built-in
// BackupStorageClasses in root types, so controller-gen renders their
// kubebuilder markers into an OpenAPI schema. The leading underscore keeps the
// package out of ./... and therefore out of config/crd.
//
// +groupName=schema.backupstorageclasses.openeverest.io
// +versionName=v1
package schema

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
)

// S3 carries the parameters of the s3 class.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type S3 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec storageclasses.S3 `json:"spec"`
}
