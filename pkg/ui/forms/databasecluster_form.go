// Package forms provides UI form definitions for OpenEverest
package forms

import (
	corev1 "k8s.io/api/core/v1"

	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
)

// DatabaseClusterForm defines the UI form for DatabaseCluster
type DatabaseClusterForm struct {
	// TLS configuration
	TLS *TLSSpecForm `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// TLSSpecForm defines the UI form for TLSSpec
type TLSSpecForm struct {
	// Mode: require | verify-ca | verify-full
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Secret references
	CASecretRef         *SecretRefForm `json:"caSecretRef,omitempty" yaml:"caSecretRef,omitempty"`
	ServerCertSecretRef *SecretRefForm `json:"serverCertSecretRef,omitempty" yaml:"serverCertSecretRef,omitempty"`
	ClientCertSecretRef *SecretRefForm `json:"clientCertSecretRef,omitempty" yaml:"clientCertSecretRef,omitempty"`
}

// SecretRefForm defines a secret reference in UI forms
type SecretRefForm struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Key  string `json:"key,omitempty" yaml:"key,omitempty"`
}

// ToTLSSpec converts a TLSSpecForm to a TLSSpec
func (f *TLSSpecForm) ToTLSSpec() *everestv1alpha1.TLSSpec {
	if f == nil {
		return nil
	}

	tls := &everestv1alpha1.TLSSpec{
		Mode: f.Mode,
	}

	if f.CASecretRef != nil {
		tls.CASecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: f.CASecretRef.Name},
			Key:                  f.CASecretRef.Key,
		}
	}

	if f.ServerCertSecretRef != nil {
		tls.ServerCertSecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: f.ServerCertSecretRef.Name},
			Key:                  f.ServerCertSecretRef.Key,
		}
	}

	if f.ClientCertSecretRef != nil {
		tls.ClientCertSecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: f.ClientCertSecretRef.Name},
			Key:                  f.ClientCertSecretRef.Key,
		}
	}

	return tls
}

// FromTLSSpec converts a TLSSpec to a TLSSpecForm
func FromTLSSpec(tls *everestv1alpha1.TLSSpec) *TLSSpecForm {
	if tls == nil {
		return nil
	}

	form := &TLSSpecForm{
		Mode: tls.Mode,
	}

	if tls.CASecretRef != nil {
		form.CASecretRef = &SecretRefForm{
			Name: tls.CASecretRef.Name,
			Key:  tls.CASecretRef.Key,
		}
	}

	if tls.ServerCertSecretRef != nil {
		form.ServerCertSecretRef = &SecretRefForm{
			Name: tls.ServerCertSecretRef.Name,
			Key:  tls.ServerCertSecretRef.Key,
		}
	}

	if tls.ClientCertSecretRef != nil {
		form.ClientCertSecretRef = &SecretRefForm{
			Name: tls.ClientCertSecretRef.Name,
			Key:  tls.ClientCertSecretRef.Key,
		}
	}

	return form
}
