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

package v1alpha1

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// BackupStorageClassSpec describes one object storage technology: how a
// BackupStorage of the technology is reached and how it authenticates.
type BackupStorageClassSpec struct {
	// DisplayName is the human-readable name of the technology.
	// +kubebuilder:validation:MaxLength=63
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// Description of the technology.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`
	// ParametersSchema validates BackupStorage.spec.parameters, the
	// connection parameters of the technology. The bucket and prefix are
	// BackupStorage.spec.location and transport trust is
	// BackupStorage.spec.tls, so neither belongs in it.
	// +optional
	ParametersSchema common.ParametersSchema `json:"parametersSchema,omitempty"`
	// AuthModes lists the authentication modes of the technology, each with
	// the credentials Secret keys it needs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +listType=map
	// +listMapKey=name
	AuthModes []common.AuthMode `json:"authModes"`
	// UISchema holds rendering hints for the forms that create a
	// BackupStorage of the class, grouped by form: "location" for
	// spec.location, "storage" for spec.parameters and "credentials" keyed by
	// authentication mode.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`
}

// FindAuthMode returns the class's authentication mode named name, or nil.
func (s *BackupStorageClassSpec) FindAuthMode(name string) *common.AuthMode {
	i := slices.IndexFunc(s.AuthModes, func(m common.AuthMode) bool { return m.Name == name })
	if i < 0 {
		return nil
	}
	return &s.AuthModes[i]
}

// AuthModeNames returns the names of the class's authentication modes.
func (s *BackupStorageClassSpec) AuthModeNames() []string {
	names := make([]string, 0, len(s.AuthModes))
	for _, m := range s.AuthModes {
		names = append(names, m.Name)
	}
	return names
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=bsc
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BackupStorageClass describes an object storage technology that
// BackupStorages reference. It is pure data: the backup classes that can write
// to it declare so in BackupClass.spec.backupStorageClasses.
type BackupStorageClass struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec BackupStorageClassSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// BackupStorageClassList contains a list of BackupStorageClass.
type BackupStorageClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []BackupStorageClass `json:"items"`
}

func init() { //nolint:gochecknoinits // kubebuilder scheme registration convention
	SchemeBuilder.Register(&BackupStorageClass{}, &BackupStorageClassList{})
}
