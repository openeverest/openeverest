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

package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"

	"github.com/percona/everest/cmd/config"
	"github.com/percona/everest/pkg/session"
)

func TestHTTPServerTimeouts_PlainHTTP(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	logger := zap.NewNop().Sugar()
	e := &EverestServer{
		config:     &config.EverestConfig{},
		l:          logger,
		echo:       echo.New(),
		sessionMgr: &session.Manager{},
	}

	err := e.initHTTPServer(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5*time.Second, e.echo.Server.ReadHeaderTimeout, "ReadHeaderTimeout should be 5s")
	assert.Equal(t, 60*time.Second, e.echo.Server.ReadTimeout, "ReadTimeout should be 60s")
	assert.Equal(t, 120*time.Second, e.echo.Server.IdleTimeout, "IdleTimeout should be 120s")
	assert.Equal(t, time.Duration(0), e.echo.Server.WriteTimeout, "WriteTimeout must remain 0 to avoid breaking streaming responses")
}

func TestHTTPServerTimeouts_TLS(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	createTestTLSCert(t, tempDir)

	watcher, err := certwatcher.New(
		filepath.Join(tempDir, "tls.crt"),
		filepath.Join(tempDir, "tls.key"),
	)
	require.NoError(t, err)

	e := &EverestServer{
		config: &config.EverestConfig{
			TLSCertsPath: tempDir,
		},
		l: zap.NewNop().Sugar(),
	}

	server := e.newTLSServer("127.0.0.1:8443", watcher)
	require.NotNil(t, server)

	assert.Equal(t, 5*time.Second, server.ReadHeaderTimeout, "ReadHeaderTimeout should be 5s")
	assert.Equal(t, 60*time.Second, server.ReadTimeout, "ReadTimeout should be 60s")
	assert.Equal(t, 120*time.Second, server.IdleTimeout, "IdleTimeout should be 120s")
	assert.Equal(t, time.Duration(0), server.WriteTimeout, "WriteTimeout must remain 0 to avoid breaking streaming responses")
	assert.NotNil(t, server.TLSConfig)
	assert.NotNil(t, server.TLSConfig.GetCertificate)
}

func createTestTLSCert(t *testing.T, dir string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"OpenEverest Test"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPath := filepath.Clean(filepath.Join(dir, "tls.crt"))
	err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	keyPath := filepath.Clean(filepath.Join(dir, "tls.key"))
	err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)
	require.NoError(t, err)
}
