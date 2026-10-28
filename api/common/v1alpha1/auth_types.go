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

import "fmt"

// Authentication modes reserved by core. They are the only modes without a
// credentials Secret: every mode a class defines is Secret-backed. The
// XValidation rules on AuthMode and Authentication spell the same names and
// must stay identical to IsReservedAuthMode.
const (
	// AuthModeWorkloadIdentity authenticates as the identity of the pod that
	// talks to the service (IRSA, EKS Pod Identity, GKE or Azure Workload
	// Identity).
	AuthModeWorkloadIdentity = "WorkloadIdentity"
	// AuthModeAnonymous sends no credentials.
	AuthModeAnonymous = "Anonymous"
)

// IsReservedAuthMode reports whether name is a mode reserved by core, which
// takes no credentials Secret.
func IsReservedAuthMode(name string) bool {
	return name == AuthModeWorkloadIdentity || name == AuthModeAnonymous
}

// AuthMode is one authentication method a class supports. A mode the class
// defines lists the credentials Secret keys it needs; the reserved modes
// (WorkloadIdentity, Anonymous) list none.
//
// +kubebuilder:validation:XValidation:rule="(self.name in ['WorkloadIdentity', 'Anonymous']) != has(self.credentials)",message="WorkloadIdentity and Anonymous take no credentials; every other mode must declare its credentials keys"
type AuthMode struct {
	// Name identifies the mode, in PascalCase (e.g. AccessKey, BasicAuth).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Z][A-Za-z0-9]{0,62}$`
	Name string `json:"name"`
	// Credentials lists the keys the credentials Secret must or may carry.
	// +optional
	Credentials *CredentialKeys `json:"credentials,omitempty"`
}

// CredentialKeys lists the keys of a credentials Secret.
type CredentialKeys struct {
	// Required keys must be present and non-empty.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:items:MaxLength=253
	// +kubebuilder:validation:items:Pattern=`^[-._a-zA-Z0-9]+$`
	// +listType=set
	Required []string `json:"required"`
	// Optional keys are read when present.
	// +optional
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:items:MaxLength=253
	// +kubebuilder:validation:items:Pattern=`^[-._a-zA-Z0-9]+$`
	// +listType=set
	Optional []string `json:"optional,omitempty"`
}

// ValidateCredentials checks that data carries every key the mode requires.
// Keys the mode does not list are ignored.
func (m *AuthMode) ValidateCredentials(data map[string][]byte) error {
	if m.Credentials == nil {
		return nil
	}
	for _, key := range m.Credentials.Required {
		if len(data[key]) == 0 {
			return fmt.Errorf("missing key %q required by authentication mode %q", key, m.Name)
		}
	}
	return nil
}

// Authentication names how an object authenticates to the service it
// references: a mode of its class and, for every mode but the reserved ones,
// the Secret holding that mode's credentials.
//
// +kubebuilder:validation:XValidation:rule="(self.mode in ['WorkloadIdentity', 'Anonymous']) != has(self.credentialsSecretRef)",message="credentialsSecretRef is required for every mode except WorkloadIdentity and Anonymous, which take none"
type Authentication struct {
	// Mode names one of the class's authModes.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Z][A-Za-z0-9]{0,62}$`
	Mode string `json:"mode"`
	// CredentialsSecretRef references the Secret in the same namespace that
	// holds the credentials keys of the mode.
	// +optional
	CredentialsSecretRef *SecretRef `json:"credentialsSecretRef,omitempty"`
}
