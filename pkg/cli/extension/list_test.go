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
)

func TestPluginLister_Render_JSON(t *testing.T) {
	t.Parallel()

	extList := []Info{
		{
			Name:        "my-extension",
			DisplayName: "My Extension",
			BackendURL:  "my-service.everest-system:8080 (in-cluster)",
			Enabled:     true,
		},
		{
			Name:        "monitoring-ext",
			DisplayName: "Monitoring Extension",
			BackendURL:  "https://ext.example.com",
			Enabled:     false,
		},
	}

	lister := &PluginLister{
		cfg: ListConfig{Pretty: false},
	}

	var buf bytes.Buffer
	lister.Render(&buf, extList)

	assert.JSONEq(t, `[{"name":"my-extension","displayName":"My Extension","backendUrl":"my-service.everest-system:8080 (in-cluster)","enabled":true},{"name":"monitoring-ext","displayName":"Monitoring Extension","backendUrl":"https://ext.example.com","enabled":false}]`, buf.String())

	var decoded []Info
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	assert.Equal(t, "my-extension", decoded[0].Name)
	assert.Equal(t, "My Extension", decoded[0].DisplayName)
	assert.Equal(t, "my-service.everest-system:8080 (in-cluster)", decoded[0].BackendURL)
	assert.True(t, decoded[0].Enabled)
	assert.Equal(t, "monitoring-ext", decoded[1].Name)
	assert.Equal(t, "Monitoring Extension", decoded[1].DisplayName)
	assert.Equal(t, "https://ext.example.com", decoded[1].BackendURL)
	assert.False(t, decoded[1].Enabled)
}

func TestPluginLister_Render_JSON_Empty(t *testing.T) {
	t.Parallel()

	lister := &PluginLister{
		cfg: ListConfig{Pretty: false},
	}

	// Empty slice should output "[]\n"
	var buf bytes.Buffer
	lister.Render(&buf, []Info{})
	assert.Equal(t, "[]\n", buf.String())

	var decoded []Info
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)

	// nil slice should also output "[]\n", not "null\n"
	buf.Reset()
	lister.Render(&buf, nil)
	assert.Equal(t, "[]\n", buf.String())
}

func TestPluginLister_Render_Table(t *testing.T) {
	t.Parallel()

	extList := []Info{
		{
			Name:        "my-extension",
			DisplayName: "My Extension",
			BackendURL:  "my-service.everest-system:8080 (in-cluster)",
			Enabled:     true,
		},
		{
			Name:        "monitoring-ext",
			DisplayName: "Monitoring Extension",
			BackendURL:  "https://ext.example.com",
			Enabled:     false,
		},
	}

	lister := &PluginLister{
		cfg: ListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, extList)

	out := buf.String()
	// Headers
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "DISPLAY NAME")
	assert.Contains(t, out, "BACKEND URL")
	assert.Contains(t, out, "ENABLED")

	// Rows
	assert.Contains(t, out, "my-extension")
	assert.Contains(t, out, "My Extension")
	assert.Contains(t, out, "my-service.everest-system:8080 (in-cluster)")
	assert.Contains(t, out, "true")

	assert.Contains(t, out, "monitoring-ext")
	assert.Contains(t, out, "Monitoring Extension")
	assert.Contains(t, out, "https://ext.example.com")
	assert.Contains(t, out, "false")
}

func TestPluginLister_Render_Table_Empty(t *testing.T) {
	t.Parallel()

	lister := &PluginLister{
		cfg: ListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, []Info{})

	out := buf.String()
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "DISPLAY NAME")
	assert.Contains(t, out, "BACKEND URL")
	assert.Contains(t, out, "ENABLED")

	// Only header should be present, no data rows
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
}
