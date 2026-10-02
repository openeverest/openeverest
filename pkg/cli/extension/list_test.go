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

package extension

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

func TestPluginLister_Render_JSON(t *testing.T) {
	t.Parallel()

	plugins := []PluginInfo{
		{
			Name:        "backup-storage",
			DisplayName: "Backup Storage Extension",
			BackendURL:  "backup-storage.everest:8080 (in-cluster)",
			Enabled:     true,
		},
		{
			Name:        "monitoring",
			DisplayName: "Monitoring Plugin",
			BackendURL:  "https://monitoring.example.com",
			Enabled:     false,
		},
	}

	lister := &PluginLister{
		cfg: ListConfig{Pretty: false},
	}

	var buf bytes.Buffer
	lister.render(&buf, plugins)

	expectedJSON := `[
		{
			"name": "backup-storage",
			"displayName": "Backup Storage Extension",
			"backendUrl": "backup-storage.everest:8080 (in-cluster)",
			"enabled": true
		},
		{
			"name": "monitoring",
			"displayName": "Monitoring Plugin",
			"backendUrl": "https://monitoring.example.com",
			"enabled": false
		}
	]`
	assert.JSONEq(t, expectedJSON, buf.String())

	var decoded []PluginInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	assert.Equal(t, plugins, decoded)
}

func TestPluginLister_Render_JSON_Empty(t *testing.T) {
	t.Parallel()

	lister := &PluginLister{
		cfg: ListConfig{Pretty: false},
	}

	// Empty slice should output "[]\n"
	var buf bytes.Buffer
	lister.render(&buf, []PluginInfo{})
	assert.Equal(t, "[]\n", buf.String())

	var decoded []PluginInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)

	// nil slice should also output "[]\n", not "null\n"
	buf.Reset()
	lister.render(&buf, nil)
	assert.Equal(t, "[]\n", buf.String())
}

func TestPluginLister_Render_Table(t *testing.T) {
	t.Parallel()

	plugins := []PluginInfo{
		{
			Name:        "backup-storage",
			DisplayName: "Backup Storage Extension",
			BackendURL:  "backup-storage.everest:8080 (in-cluster)",
			Enabled:     true,
		},
		{
			Name:        "monitoring",
			DisplayName: "Monitoring Plugin",
			BackendURL:  "https://monitoring.example.com",
			Enabled:     false,
		},
	}

	lister := &PluginLister{
		cfg: ListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.render(&buf, plugins)

	out := buf.String()
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "DISPLAY NAME")
	assert.Contains(t, out, "BACKEND URL")
	assert.Contains(t, out, "ENABLED")

	assert.Contains(t, out, "backup-storage")
	assert.Contains(t, out, "Backup Storage Extension")
	assert.Contains(t, out, "backup-storage.everest:8080 (in-cluster)")
	assert.Contains(t, out, "true")

	assert.Contains(t, out, "monitoring")
	assert.Contains(t, out, "Monitoring Plugin")
	assert.Contains(t, out, "https://monitoring.example.com")
	assert.Contains(t, out, "false")
}

func TestPluginLister_Render_Table_Empty(t *testing.T) {
	t.Parallel()

	lister := &PluginLister{
		cfg: ListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.render(&buf, []PluginInfo{})

	out := buf.String()
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "DISPLAY NAME")
	assert.Contains(t, out, "BACKEND URL")
	assert.Contains(t, out, "ENABLED")

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
}

func TestFormatBackendURL(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatBackendURL(nil))
	assert.Empty(t, formatBackendURL(&extensionsv1alpha1.PluginSpec{}))

	inCluster := &extensionsv1alpha1.PluginSpec{
		Backend: &extensionsv1alpha1.PluginBackend{
			ServiceRef: &extensionsv1alpha1.PluginBackendServiceRef{
				Namespace: "everest",
				Name:      "my-svc",
				Port:      8080,
			},
		},
	}
	assert.Equal(t, "my-svc.everest:8080 (in-cluster)", formatBackendURL(inCluster))

	external := &extensionsv1alpha1.PluginSpec{
		Backend: &extensionsv1alpha1.PluginBackend{
			ExternalURL: "https://example.com/api",
		},
	}
	assert.Equal(t, "https://example.com/api", formatBackendURL(external))
}
