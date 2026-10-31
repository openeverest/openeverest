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

package storageclasses

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// SecretContent is a Secret as Accept sees it.
type SecretContent struct {
	// Found reports whether the Secret exists.
	Found bool
	// Data is the Secret's data.
	Data map[string][]byte
}

// AcceptInput is what Accept judges: a BackupStorage's spec and the content of
// the Secrets it references.
type AcceptInput struct {
	Location       backupv1alpha1.BackupStorageLocation
	Parameters     *runtime.RawExtension
	Authentication common.Authentication
	TLS            *common.TLSConfig
	// Credentials is the Secret named by Authentication.CredentialsSecretRef.
	Credentials SecretContent
	// CA is the Secret named by TLS.CA.SecretKeyRef.
	CA SecretContent
}

// Verdict is the outcome of Accept.
type Verdict struct {
	// Reason is backupv1alpha1.ReasonAccepted, or why the storage is not
	// accepted.
	Reason string
	// Message explains the reason.
	Message string
}

// Accepted reports whether the storage is valid for its class.
func (v Verdict) Accepted() bool {
	return v.Reason == backupv1alpha1.ReasonAccepted
}

// Accept judges a BackupStorage against its class. It reads nothing but its
// arguments, so the controller, list filters and executors that check a
// storage before using it reach the same verdict.
func Accept(class *backupv1alpha1.BackupStorageClass, in AcceptInput) Verdict {
	reject := func(reason, format string, args ...any) Verdict {
		return Verdict{Reason: reason, Message: fmt.Sprintf(format, args...)}
	}

	if msg := classProblem(class); msg != "" {
		return reject(backupv1alpha1.ReasonInvalidClass, "BackupStorageClass %q %s", class.Name, msg)
	}
	params := in.Parameters
	// An absent payload must still satisfy the schema's required properties.
	if params == nil && class.Spec.ParametersSchema.OpenAPIV3Schema != nil {
		params = &runtime.RawExtension{Raw: []byte("{}")}
	}
	if err := class.Spec.ParametersSchema.Validate(params); err != nil {
		return reject(backupv1alpha1.ReasonInvalidParameters,
			"parameters rejected by the parametersSchema of BackupStorageClass %q: %v", class.Name, err,
		)
	}
	if v, ok := acceptBuiltin(class.Name, in); !ok {
		return v
	}

	mode := class.Spec.FindAuthMode(in.Authentication.Mode)
	if mode == nil {
		return reject(backupv1alpha1.ReasonAuthModeUnsupported, "BackupStorageClass %q has no authentication mode %q; it supports %s",
			class.Name, in.Authentication.Mode, strings.Join(class.Spec.AuthModeNames(), ", "),
		)
	}
	if ref := in.Authentication.CredentialsSecretRef; ref != nil {
		if !in.Credentials.Found {
			return reject(backupv1alpha1.ReasonCredentialsNotFound, "credentials Secret %q not found", ref.Name)
		}
		if err := mode.ValidateCredentials(in.Credentials.Data); err != nil {
			return reject(backupv1alpha1.ReasonCredentialsNotFound, "credentials Secret %q: %v", ref.Name, err)
		}
	}
	if in.TLS != nil && in.TLS.CA != nil && in.TLS.CA.SecretKeyRef != nil {
		ref := in.TLS.CA.SecretKeyRef
		if !in.CA.Found {
			return reject(backupv1alpha1.ReasonInvalidCABundle, "CA Secret %q not found", ref.Name)
		}
		if _, err := common.ParseCABundle(in.CA.Data[ref.Key]); err != nil {
			return reject(backupv1alpha1.ReasonInvalidCABundle, "CA Secret %q key %q: %v", ref.Name, ref.Key, err)
		}
	}
	return Verdict{Reason: backupv1alpha1.ReasonAccepted, Message: fmt.Sprintf("valid for BackupStorageClass %q", class.Name)}
}

// classProblem describes a rule every class must follow that class breaks,
// or returns "".
func classProblem(class *backupv1alpha1.BackupStorageClass) string {
	if schema := class.Spec.ParametersSchema.OpenAPIV3Schema; schema != nil {
		for _, name := range []string{"bucket", "prefix"} {
			if _, ok := schema.Properties[name]; ok {
				return fmt.Sprintf("declares %q as a parameter; it is BackupStorage spec.location", name)
			}
		}
	}
	for _, m := range class.Spec.AuthModes {
		if common.IsReservedAuthMode(m.Name) != (m.Credentials == nil) {
			return fmt.Sprintf("declares authentication mode %q with credentials keys only modes other than %s and %s may have",
				m.Name, common.AuthModeWorkloadIdentity, common.AuthModeAnonymous,
			)
		}
	}
	return ""
}

// acceptBuiltin applies the decoder of a built-in class, which knows rules
// the schema cannot express.
func acceptBuiltin(className string, in AcceptInput) (Verdict, bool) {
	if className != ClassS3 {
		return Verdict{}, true
	}
	if err := validateS3Location(in.Location); err != nil {
		return Verdict{Reason: backupv1alpha1.ReasonInvalidLocation, Message: err.Error()}, false
	}
	s3, err := decodeS3(in.Parameters)
	if err != nil {
		return Verdict{Reason: backupv1alpha1.ReasonInvalidParameters, Message: err.Error()}, false
	}
	if ep, _ := s3.Endpoint(); !ep.Secure && in.TLS != nil && (in.TLS.CA != nil || in.TLS.InsecureSkipVerify) {
		return Verdict{
			Reason:  backupv1alpha1.ReasonInvalidParameters,
			Message: fmt.Sprintf("spec.tls is set but endpointURL %q is not https", s3.EndpointURL),
		}, false
	}
	return Verdict{}, true
}
