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

package s3

// StorageView is the spec of an s3 BackupStorage as the class's validations
// see it: self in each rule. It has the JSON shape of BackupStorage.spec,
// with parameters typed as Parameters, and carries the rules as XValidation
// markers, which make gen writes into the class's validations. Each marker
// names in fieldPath the field it judges: a rule on .location reports
// InvalidLocation, any other InvalidParameters.
//
// +kubebuilder:validation:XValidation:rule="has(self.location.objectStore) && self.location.objectStore.bucket.matches('^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$') && !self.location.objectStore.bucket.contains('..')",message="an s3 bucket name must be 3 to 63 lowercase letters, digits, '.', '-' or '_', start and end with a letter or digit, and contain no '..'",fieldPath=".location.objectStore.bucket"
// +kubebuilder:validation:XValidation:rule="!(has(self.tls) && (has(self.tls.ca) || (has(self.tls.insecureSkipVerify) && self.tls.insecureSkipVerify)) && has(self.parameters) && has(self.parameters.endpointURL) && self.parameters.endpointURL.startsWith('http://'))",message="tls applies to https endpoints only, and endpointURL is http",fieldPath=".tls"
type StorageView struct {
	// +kubebuilder:validation:Required
	Location StorageLocation `json:"location"`
	// +optional
	Parameters *Parameters `json:"parameters,omitempty"`
	// +kubebuilder:validation:Required
	Authentication StorageAuthentication `json:"authentication"`
	// +optional
	TLS *StorageTLS `json:"tls,omitempty"`
}

// StorageLocation is BackupStorage.spec.location.
type StorageLocation struct {
	// +optional
	ObjectStore *ObjectStore `json:"objectStore,omitempty"`
}

// ObjectStore is BackupStorage.spec.location.objectStore.
type ObjectStore struct {
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// StorageAuthentication is BackupStorage.spec.authentication.
type StorageAuthentication struct {
	// +kubebuilder:validation:Required
	Mode string `json:"mode"`
	// +optional
	CredentialsSecretRef *LocalObjectReference `json:"credentialsSecretRef,omitempty"`
}

// StorageTLS is BackupStorage.spec.tls.
type StorageTLS struct {
	// +optional
	CA *CABundleSource `json:"ca,omitempty"`
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// CABundleSource is BackupStorage.spec.tls.ca.
type CABundleSource struct {
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// LocalObjectReference names an object in the storage's namespace.
type LocalObjectReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// SecretKeySelector is a key of a Secret in the storage's namespace.
type SecretKeySelector struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}
