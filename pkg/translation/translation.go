// Package translation provides translation utilities for OpenEverest
package translation

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
)

// TranslateTLS translates TLS configuration from shared to provider-specific
func TranslateTLS(src *everestv1alpha1.TLSSpec) *everestv1alpha1.TLSSpec {
	if src == nil {
		return nil
	}

	dst := &everestv1alpha1.TLSSpec{
		Mode: src.Mode,
	}

	// Copy secret refs
	if src.CASecretRef != nil {
		dst.CASecretRef = src.CASecretRef.DeepCopy()
	}
	if src.ServerCertSecretRef != nil {
		dst.ServerCertSecretRef = src.ServerCertSecretRef.DeepCopy()
	}
	if src.ClientCertSecretRef != nil {
		dst.ClientCertSecretRef = src.ClientCertSecretRef.DeepCopy()
	}

	return dst
}

// ValidateTLSForProvider validates TLS configuration for a specific provider
func ValidateTLSForProvider(tls *everestv1alpha1.TLSSpec, provider string) field.ErrorList {
	var allErrs field.ErrorList

	if tls == nil {
		return allErrs
	}

	// Provider-specific TLS validation
	switch provider {
	case "postgresql":
		// PostgreSQL supports all TLS modes
		if tls.Mode != "" && tls.Mode != "require" && tls.Mode != "verify-ca" && tls.Mode != "verify-full" {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("tls").Child("mode"),
				tls.Mode, "PostgreSQL only supports: require, verify-ca, verify-full"))
		}
		// PostgreSQL supports CA and server certs
		if tls.Mode == "verify-ca" || tls.Mode == "verify-full" {
			if tls.CASecretRef == nil {
				allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("tls").Child("caSecretRef"),
					"required for verify-ca and verify-full modes in PostgreSQL"))
			}
		}
		if tls.Mode == "verify-full" {
			if tls.ServerCertSecretRef == nil {
				allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("tls").Child("serverCertSecretRef"),
					"required for verify-full mode in PostgreSQL"))
			}
		}

	case "psmdb":
		// MongoDB supports require and verify-ca, but not verify-full
		if tls.Mode != "" {
			if tls.Mode != "require" && tls.Mode != "verify-ca" {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("tls").Child("mode"),
					tls.Mode, "MongoDB only supports: require, verify-ca"))
			}
		}
		// MongoDB supports CA
		if tls.Mode == "verify-ca" {
			if tls.CASecretRef == nil {
				allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("tls").Child("caSecretRef"),
					"required for verify-ca mode in MongoDB"))
			}
		}

	case "pxc":
		// Percona XtraDB Cluster supports require and verify-ca
		if tls.Mode != "" {
			if tls.Mode != "require" && tls.Mode != "verify-ca" {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("tls").Child("mode"),
					tls.Mode, "PXC only supports: require, verify-ca"))
			}
		}
		// PXC supports CA
		if tls.Mode == "verify-ca" {
			if tls.CASecretRef == nil {
				allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("tls").Child("caSecretRef"),
					"required for verify-ca mode in PXC"))
			}
		}

	default:
		// Unknown provider - reject
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("engine").Child("type"),
			provider, "unsupported provider for TLS configuration"))
	}

	return allErrs
}

// ValidateSecretDataKeys validates that referenced secrets contain required keys according to contract
func ValidateSecretDataKeys(secretData map[string][]byte, refType string, path *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	switch refType {
	case "caSecretRef":
		if _, ok := secretData["ca.crt"]; !ok {
			allErrs = append(allErrs, field.Required(path, "secret data must contain 'ca.crt' key"))
		}
	case "serverCertSecretRef":
		if _, ok := secretData["tls.crt"]; !ok {
			allErrs = append(allErrs, field.Required(path, "secret data must contain 'tls.crt' key"))
		}
		if _, ok := secretData["tls.key"]; !ok {
			allErrs = append(allErrs, field.Required(path, "secret data must contain 'tls.key' key"))
		}
	case "clientCertSecretRef":
		if _, ok := secretData["tls.crt"]; !ok {
			allErrs = append(allErrs, field.Required(path, "secret data must contain 'tls.crt' key"))
		}
		if _, ok := secretData["tls.key"]; !ok {
			allErrs = append(allErrs, field.Required(path, "secret data must contain 'tls.key' key"))
		}
	default:
		allErrs = append(allErrs, field.Invalid(path, refType, "unknown secret ref type"))
	}

	return allErrs
}

// ValidateCustomSpecConflict checks for conflicts between spec.tls and custom engine config TLS settings
func ValidateCustomSpecConflict(dc *everestv1alpha1.DatabaseCluster) field.ErrorList {
	var allErrs field.ErrorList

	if dc == nil || dc.Spec == nil || dc.Spec.TLS == nil {
		return allErrs
	}

	if dc.Spec.Engine.Config != nil && *dc.Spec.Engine.Config != "" {
		configStr := strings.ToLower(*dc.Spec.Engine.Config)
		conflictKeywords := []string{"ssl =", "sslmode =", "ssl-ca =", "ssl-cert =", "ssl-key =", "tls ="}
		for _, kw := range conflictKeywords {
			if strings.Contains(configStr, kw) {
				allErrs = append(allErrs, field.Forbidden(
					field.NewPath("spec").Child("engine").Child("config"),
					"cannot configure custom TLS options in engine config when spec.tls is set",
				))
				break
			}
		}
	}

	return allErrs
}
