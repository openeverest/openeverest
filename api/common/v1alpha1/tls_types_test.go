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

package v1alpha1_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func TestParseCABundle(t *testing.T) {
	t.Parallel()

	ca := selfSignedPEM(t, "test-ca")
	other := selfSignedPEM(t, "other-ca")
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("ignored")})

	tests := []struct {
		name      string
		bundle    []byte
		wantCerts int
		wantErr   error
	}{
		{name: "one certificate", bundle: ca, wantCerts: 1},
		{name: "two certificates", bundle: append(append([]byte{}, ca...), other...), wantCerts: 2},
		{name: "non-certificate blocks are ignored", bundle: append(append([]byte{}, key...), ca...), wantCerts: 1},
		{name: "empty", bundle: nil, wantErr: common.ErrNoCertificates},
		{name: "not PEM", bundle: []byte("not a bundle"), wantErr: common.ErrNoCertificates},
		{name: "only a key", bundle: key, wantErr: common.ErrNoCertificates},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool, err := common.ParseCABundle(tt.bundle)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, pool.Subjects(), tt.wantCerts) //nolint:staticcheck // Subjects is fine for a pool built from PEM
		})
	}

	t.Run("malformed certificate", func(t *testing.T) {
		t.Parallel()

		_, err := common.ParseCABundle(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("garbage")}))
		require.ErrorContains(t, err, "parse certificate")
	})
}

func selfSignedPEM(t *testing.T, cn string) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
