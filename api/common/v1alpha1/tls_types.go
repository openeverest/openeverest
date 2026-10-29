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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// TLSConfig is how a client verifies the server it connects to. Without a CA
// the system trust store is used.
//
// +kubebuilder:validation:XValidation:rule="!(has(self.ca) && has(self.insecureSkipVerify) && self.insecureSkipVerify)",message="ca and insecureSkipVerify are mutually exclusive"
type TLSConfig struct {
	// CA is the PEM bundle the server certificate is verified against,
	// instead of the system trust store.
	// +optional
	CA *CABundleSource `json:"ca,omitempty"`
	// InsecureSkipVerify disables verification of the server certificate.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// CABundleSource selects where a PEM CA bundle is read from.
//
// +kubebuilder:validation:MinProperties=1
type CABundleSource struct {
	// SecretKeyRef selects the Secret key holding the PEM bundle.
	// +optional
	SecretKeyRef *SecretKeyRef `json:"secretKeyRef,omitempty"`
}

// SecretKeyRef selects one key of a Secret in the same namespace as the
// referrer.
//
// +structType=atomic
type SecretKeyRef struct {
	// Name of the referenced Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	Name string `json:"name"`
	// Key within the Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[-._a-zA-Z0-9]+$`
	Key string `json:"key"`
}

// ErrNoCertificates is returned by ParseCABundle for a bundle without a
// certificate.
var ErrNoCertificates = errors.New("no PEM certificate found")

// ParseCABundle parses a PEM bundle into a certificate pool. Every
// CERTIFICATE block must parse; other blocks are ignored.
func ParseCABundle(bundle []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	found := false
	for rest := bundle; ; {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate: %w", err)
		}
		pool.AddCert(cert)
		found = true
	}
	if !found {
		return nil, ErrNoCertificates
	}
	return pool, nil
}
