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

package plugintoken

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

func auditPlugin() *extensionsv1alpha1.Plugin {
	return &extensionsv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "audit", UID: "uid-1"},
		Spec: extensionsv1alpha1.PluginSpec{
			Enabled: true,
			Permissions: []extensionsv1alpha1.PluginPermission{
				{Verb: "read-events", Resource: "instances"},
				{Verb: "read", Resource: "platform-events"},
			},
		},
	}
}

// mintAndParse mints a token for plugin and parses it the way the API's JWT
// middleware does, into map claims.
func mintAndParse(t *testing.T, plugin *extensionsv1alpha1.Plugin) *Claims {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	raw, err := NewServiceWithKey(key).Mint(plugin, time.Now(), time.Hour)
	require.NoError(t, err)

	token, err := jwt.Parse(raw, func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
	require.NoError(t, err)
	claims, err := ClaimsOf(token)
	require.NoError(t, err)
	return claims
}

func TestMintedTokenCarriesThePlugin(t *testing.T) {
	t.Parallel()

	plugin := auditPlugin()
	claims := mintAndParse(t, plugin)

	assert.Equal(t, "plugin:audit", claims.Subject)
	assert.Equal(t, "audit", claims.PluginName())
	assert.Equal(t, "uid-1", claims.PluginUID)
	assert.Equal(t, plugin.Spec.Permissions, claims.Permissions)
	// The session blocklist keys tokens by their ID.
	assert.NotEmpty(t, claims.ID)
}

func TestClaimsOfRejectsUserTokens(t *testing.T) {
	t.Parallel()

	_, err := ClaimsOf(&jwt.Token{Claims: jwt.MapClaims{"sub": "admin:login"}})
	assert.Error(t, err)
}

func TestValidFor(t *testing.T) {
	t.Parallel()

	claims := mintAndParse(t, auditPlugin())
	testCases := []struct {
		desc   string
		change func(p *extensionsv1alpha1.Plugin)
		want   bool
	}{
		{"the plugin it was issued to", func(*extensionsv1alpha1.Plugin) {}, true},
		{"a permission was added", func(p *extensionsv1alpha1.Plugin) {
			p.Spec.Permissions = append(p.Spec.Permissions, extensionsv1alpha1.PluginPermission{Verb: "read", Resource: "backups"})
		}, true},
		{"a permission was dropped", func(p *extensionsv1alpha1.Plugin) { p.Spec.Permissions = p.Spec.Permissions[:1] }, false},
		{"the plugin was re-created", func(p *extensionsv1alpha1.Plugin) { p.UID = "uid-2" }, false},
		{"the plugin is disabled", func(p *extensionsv1alpha1.Plugin) { p.Spec.Enabled = false }, false},
		{"the plugin is being deleted", func(p *extensionsv1alpha1.Plugin) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
		}, false},
		{"another plugin", func(p *extensionsv1alpha1.Plugin) { p.Name = "other" }, false},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			plugin := auditPlugin()
			tc.change(plugin)
			assert.Equal(t, tc.want, claims.ValidFor(plugin))
		})
	}
}

func TestAllows(t *testing.T) {
	t.Parallel()

	permissions := []extensionsv1alpha1.PluginPermission{
		{Verb: "read-events", Resource: "instances"},
		{Verb: "*", Resource: "backup-*"},
	}
	testCases := []struct {
		resource, action string
		want             bool
	}{
		{"instances", "read-events", true},
		{"instances", "read", false},
		{"backups", "read-events", false},
		{"backup-storages", "update", true},
		{"platform-events", "read", false},
	}
	for _, tc := range testCases {
		t.Run(tc.resource+"/"+tc.action, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Allows(permissions, tc.resource, tc.action))
		})
	}
	assert.False(t, Allows(nil, "instances", "read"))
}
