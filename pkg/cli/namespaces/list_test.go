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

package namespaces

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceLister_Render_JSON(t *testing.T) {
	t.Parallel()

	nsList := []NamespaceInfo{
		{
			Name:               "default",
			InstalledOperators: []string{},
		},
		{
			Name:               "everest",
			InstalledOperators: []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"},
		},
	}

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: false},
	}

	var buf bytes.Buffer
	lister.Render(&buf, nsList)

	assert.JSONEq(t, `[{"name":"default","installedOperators":[]},{"name":"everest","installedOperators":["psmdb(v1.16.0)","postgresql(v1.16.0)"]}]`, buf.String())

	// Verify unmarshaling into struct slice
	var decoded []NamespaceInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	assert.Equal(t, "default", decoded[0].Name)
	assert.Empty(t, decoded[0].InstalledOperators)
	assert.Equal(t, "everest", decoded[1].Name)
	assert.Equal(t, []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"}, decoded[1].InstalledOperators)
}

func TestNamespaceLister_Render_JSON_Empty(t *testing.T) {
	t.Parallel()

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: false},
	}

	// Empty slice should output "[]\n"
	var buf bytes.Buffer
	lister.Render(&buf, []NamespaceInfo{})
	assert.Equal(t, "[]\n", buf.String())

	var decoded []NamespaceInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)

	// nil slice should also output "[]\n", not "null\n"
	buf.Reset()
	lister.Render(&buf, nil)
	assert.Equal(t, "[]\n", buf.String())
}

func TestNamespaceLister_Render_Table(t *testing.T) {
	t.Parallel()

	nsList := []NamespaceInfo{
		{
			Name:               "default",
			InstalledOperators: []string{},
		},
		{
			Name:               "everest",
			InstalledOperators: []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"},
		},
	}

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, nsList)

	out := buf.String()
	// Headers
	assert.Contains(t, out, "NAMESPACE")
	assert.Contains(t, out, "MANAGED")
	assert.Contains(t, out, "OPERATORS")

	// Rows
	assert.Contains(t, out, "default")
	assert.Contains(t, out, "false")
	assert.Contains(t, out, "everest")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "psmdb(v1.16.0), postgresql(v1.16.0)")
}

func TestNamespaceLister_Render_Table_Empty(t *testing.T) {
	t.Parallel()

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, []NamespaceInfo{})

	out := buf.String()
	assert.Contains(t, out, "NAMESPACE")
	assert.Contains(t, out, "MANAGED")
	assert.Contains(t, out, "OPERATORS")
	// Only header should be present, no data rows
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
}
