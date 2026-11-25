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

package gate_test

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportsOnlyTheContract keeps the gate free of every other OpenEverest
// package and of any Kubernetes client: each SDK runs it with the client
// it already has, so the gate must not bring one, nor anything versioned
// apart from the contract. Tests may import anything.
func TestImportsOnlyTheContract(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	fset := token.NewFileSet()
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files++
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			require.NoError(t, err)
			assert.True(t, allowedImport(path), "%s imports %s", name, path)
		}
	}
	require.NotZero(t, files)
}

func allowedImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	if !strings.Contains(first, ".") {
		return true // the standard library
	}
	return path == "github.com/openeverest/openeverest/v2/contract" ||
		path == "k8s.io/apimachinery/pkg/apis/meta/v1"
}
