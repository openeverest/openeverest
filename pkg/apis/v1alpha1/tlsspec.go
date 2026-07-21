// Package v1alpha1 contains API definitions for OpenEverest
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// TLSSpec defines TLS configuration for database clusters
type TLSSpec struct {
	// Mode: require | verify-ca | verify-full
	// +kubebuilder:validation:Enum=require;verify-ca;verify-full
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// CASecretRef references the secret containing the CA certificate
	// Required for verify-ca and verify-full modes
	CASecretRef *corev1.SecretKeySelector `json:"caSecretRef,omitempty" yaml:"caSecretRef,omitempty"`

	// ServerCertSecretRef references the secret containing the server certificate and key
	// Required for verify-full mode
	ServerCertSecretRef *corev1.SecretKeySelector `json:"serverCertSecretRef,omitempty" yaml:"serverCertSecretRef,omitempty"`

	// ClientCertSecretRef references the secret containing the client certificate and key
	// Optional, used for mTLS client authentication where supported
	ClientCertSecretRef *corev1.SecretKeySelector `json:"clientCertSecretRef,omitempty" yaml:"clientCertSecretRef,omitempty"`
}

const DefaultTLSMode = "require"

// Default sets default values for TLSSpec
func (t *TLSSpec) Default() {
	if t == nil {
		return
	}
	if t.Mode == "" {
		t.Mode = DefaultTLSMode
	}
}

// Validate validates the TLSSpec
func (t *TLSSpec) Validate(path *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if t == nil {
		return allErrs
	}

	// Validate mode
	if t.Mode != "" {
		switch t.Mode {
		case "require", "verify-ca", "verify-full":
			// Valid mode
		default:
			allErrs = append(allErrs, field.Invalid(path.Child("mode"), t.Mode, "must be one of: require, verify-ca, verify-full"))
		}
	}

	// Validate secret refs if present
	if t.CASecretRef != nil {
		allErrs = append(allErrs, validateSecretKeySelector(t.CASecretRef, path.Child("caSecretRef"))...)
	}

	if t.ServerCertSecretRef != nil {
		allErrs = append(allErrs, validateSecretKeySelector(t.ServerCertSecretRef, path.Child("serverCertSecretRef"))...)
	}

	if t.ClientCertSecretRef != nil {
		allErrs = append(allErrs, validateSecretKeySelector(t.ClientCertSecretRef, path.Child("clientCertSecretRef"))...)
	}

	// Cross-field validation
	if t.Mode == "verify-ca" || t.Mode == "verify-full" {
		if t.CASecretRef == nil {
			allErrs = append(allErrs, field.Required(path.Child("caSecretRef"), "required for verify-ca and verify-full modes"))
		}
	}

	if t.Mode == "verify-full" {
		if t.ServerCertSecretRef == nil {
			allErrs = append(allErrs, field.Required(path.Child("serverCertSecretRef"), "required for verify-full mode"))
		}
	}

	return allErrs
}

func validateSecretKeySelector(s *corev1.SecretKeySelector, path *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if s.Name == "" {
		allErrs = append(allErrs, field.Required(path.Child("name"), "name is required"))
	}
	if s.Key == "" {
		allErrs = append(allErrs, field.Required(path.Child("key"), "key is required"))
	}
	return allErrs
}
