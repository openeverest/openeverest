// Package everestctl provides the CLI implementation for Everest
package everestctl

import (
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
)

// AddTLSFlags adds TLS flags to a command
func AddTLSFlags(cmd *cobra.Command) {
	cmd.Flags().String("tls-mode", "", "TLS mode: require, verify-ca, verify-full")
	cmd.Flags().String("tls-ca-secret", "", "Secret name containing CA certificate")
	cmd.Flags().String("tls-server-cert-secret", "", "Secret name containing server certificate and key")
	cmd.Flags().String("tls-client-cert-secret", "", "Secret name containing client certificate and key (for mTLS)")
}

// ParseTLSFromFlags parses TLS flags and returns a TLSSpec
func ParseTLSFromFlags(cmd *cobra.Command) *everestv1alpha1.TLSSpec {
	tls := &everestv1alpha1.TLSSpec{}

	flags := cmd.Flags()
	if mode, _ := flags.GetString("tls-mode"); mode != "" {
		tls.Mode = mode
	}

	if caSecret, _ := flags.GetString("tls-ca-secret"); caSecret != "" {
		tls.CASecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: caSecret},
			Key:                  "ca.crt", // Default key name
		}
	}

	if serverCertSecret, _ := flags.GetString("tls-server-cert-secret"); serverCertSecret != "" {
		tls.ServerCertSecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: serverCertSecret},
			Key:                  "tls.crt", // Default key name
		}
	}

	if clientCertSecret, _ := flags.GetString("tls-client-cert-secret"); clientCertSecret != "" {
		tls.ClientCertSecretRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: clientCertSecret},
			Key:                  "tls.crt", // Default key name
		}
	}

	return tls
}
