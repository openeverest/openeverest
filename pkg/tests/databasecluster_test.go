// Package tests provides tests for OpenEverest
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
	"github.com/percona/everest/pkg/translation"
	"github.com/percona/everest/pkg/validators"
)

// TestTLSSpecValidation tests TLSSpec validation
func TestTLSSpecValidation(t *testing.T) {
	// Test valid TLS configurations
	validCases := []struct {
		name string
		tls  *everestv1alpha1.TLSSpec
	}{
		{
			name: "require mode only",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "require",
			},
		},
		{
			name: "verify-ca with CA",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "verify-ca",
				CASecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "ca-secret"},
					Key:                  "ca.crt",
				},
			},
		},
		{
			name: "verify-full with CA and server cert",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "verify-full",
				CASecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "ca-secret"},
					Key:                  "ca.crt",
				},
				ServerCertSecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "server-secret"},
					Key:                  "tls.crt",
				},
			},
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.tls.Validate(field.NewPath("spec").Child("tls"))
			assert.Empty(t, errs)
		})
	}

	// Test invalid TLS configurations
	invalidCases := []struct {
		name string
		tls  *everestv1alpha1.TLSSpec
		want string
	}{
		{
			name: "invalid mode",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "invalid-mode",
			},
			want: "must be one of: require, verify-ca, verify-full",
		},
		{
			name: "verify-ca without CA",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "verify-ca",
			},
			want: "required for verify-ca and verify-full modes",
		},
		{
			name: "verify-full without server cert",
			tls: &everestv1alpha1.TLSSpec{
				Mode: "verify-full",
				CASecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "ca-secret"},
					Key:                  "ca.crt",
				},
			},
			want: "required for verify-full mode",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.tls.Validate(field.NewPath("spec").Child("tls"))
			assert.NotEmpty(t, errs)
			assert.Contains(t, errs.ToAggregate().Error(), tc.want)
		})
	}
}

// TestTLSDefaulting tests mode defaulting logic
func TestTLSDefaulting(t *testing.T) {
	tls := &everestv1alpha1.TLSSpec{}
	assert.Equal(t, "", tls.Mode)

	tls.Default()
	assert.Equal(t, "require", tls.Mode)
}

// TestSecretDataKeysValidation tests validation of secret key contracts
func TestSecretDataKeysValidation(t *testing.T) {
	// CA secret valid & invalid
	errs := translation.ValidateSecretDataKeys(map[string][]byte{"ca.crt": []byte("cert")}, "caSecretRef", field.NewPath("caSecretRef"))
	assert.Empty(t, errs)

	errs = translation.ValidateSecretDataKeys(map[string][]byte{"invalid.key": []byte("val")}, "caSecretRef", field.NewPath("caSecretRef"))
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs.ToAggregate().Error(), "ca.crt")

	// Server cert secret valid & invalid
	errs = translation.ValidateSecretDataKeys(map[string][]byte{"tls.crt": []byte("c"), "tls.key": []byte("k")}, "serverCertSecretRef", field.NewPath("serverCertSecretRef"))
	assert.Empty(t, errs)

	errs = translation.ValidateSecretDataKeys(map[string][]byte{"tls.crt": []byte("c")}, "serverCertSecretRef", field.NewPath("serverCertSecretRef"))
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs.ToAggregate().Error(), "tls.key")
}

// TestCustomSpecConflictValidation tests conflict detection between spec.tls and engine config TLS settings
func TestCustomSpecConflictValidation(t *testing.T) {
	cfg := "ssl = on\nsslmode = require"
	dc := &everestv1alpha1.DatabaseCluster{
		Spec: &everestv1alpha1.DatabaseClusterSpec{
			TLS: &everestv1alpha1.TLSSpec{
				Mode: "require",
			},
		},
	}
	dc.Spec.Engine.Config = &cfg

	errs := validators.ValidateDatabaseCluster(dc)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs.ToAggregate().Error(), "cannot configure custom TLS options in engine config when spec.tls is set")
}

// TestDatabaseClusterWithTLS tests DatabaseCluster with TLS configuration
func TestDatabaseClusterWithTLS(t *testing.T) {
	// Create a DatabaseCluster with TLS
	dc := &everestv1alpha1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
		},
		Spec: &everestv1alpha1.DatabaseClusterSpec{
			TLS: &everestv1alpha1.TLSSpec{
				Mode: "verify-full",
				CASecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "ca-secret"},
					Key:                  "ca.crt",
				},
				ServerCertSecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "server-secret"},
					Key:                  "tls.crt",
				},
			},
		},
	}
	dc.Spec.Engine.Type = everestv1alpha1.DatabaseClusterSpecEngineTypePostgresql

	// Test that the DatabaseCluster can be created
	assert.NotNil(t, dc)
	assert.Equal(t, "test-cluster", dc.Name)
	assert.Equal(t, "test-namespace", dc.Namespace)
	assert.Equal(t, everestv1alpha1.DatabaseClusterSpecEngineTypePostgresql, dc.Spec.Engine.Type)
	assert.NotNil(t, dc.Spec.TLS)
	assert.Equal(t, "verify-full", dc.Spec.TLS.Mode)
	assert.NotNil(t, dc.Spec.TLS.CASecretRef)
	assert.NotNil(t, dc.Spec.TLS.ServerCertSecretRef)
}

// TestDatabaseClusterWithoutTLS tests DatabaseCluster without TLS (backward compatibility)
func TestDatabaseClusterWithoutTLS(t *testing.T) {
	// Create a DatabaseCluster without TLS
	dc := &everestv1alpha1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
		},
		Spec: &everestv1alpha1.DatabaseClusterSpec{},
	}
	dc.Spec.Engine.Type = everestv1alpha1.DatabaseClusterSpecEngineTypePostgresql

	// Test that the DatabaseCluster can be created without TLS
	assert.NotNil(t, dc)
	assert.Equal(t, "test-cluster", dc.Name)
	assert.Equal(t, "test-namespace", dc.Namespace)
	assert.Equal(t, everestv1alpha1.DatabaseClusterSpecEngineTypePostgresql, dc.Spec.Engine.Type)
	assert.Nil(t, dc.Spec.TLS)
}
