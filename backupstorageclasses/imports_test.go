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

package backupstorageclasses_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contractsRoot is the import path of this package, the root of every
// contract.
const contractsRoot = "github.com/openeverest/openeverest/v2/backupstorageclasses"

// TestContractsImportNothingFromCore keeps the contracts free of OpenEverest
// imports, so executors outside OpenEverest can depend on them alone and a
// contract can move to a repository of its own unchanged. Tests may import
// anything.
func TestContractsImportNothingFromCore(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		files++
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			assert.True(t, allowedImport(p), "%s imports %s", path, p)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotZero(t, files)
}

// allowedImport reports whether a contract may import path: the standard
// library, the contracts, API types and their encoding, nothing that reaches
// the rest of OpenEverest.
func allowedImport(path string) bool {
	allowed := []string{
		contractsRoot,
		"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1",
		"sigs.k8s.io/yaml",
	}
	first, _, _ := strings.Cut(path, "/")
	if !strings.Contains(first, ".") {
		return true // the standard library
	}
	for _, a := range allowed {
		if path == a || strings.HasPrefix(path, a+"/") {
			return true
		}
	}
	return false
}
