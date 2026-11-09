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

// Package providerruntime_test checks invariants that span every package of
// provider-runtime.
package providerruntime_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

const (
	// runtimePath prefixes every package of provider-runtime; the functions
	// in an allowlist are named relative to it.
	runtimePath = "github.com/openeverest/openeverest/v2/provider-runtime/"
	// backupAPIPath declares the execution union of a BackupClass.
	backupAPIPath = "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// member is a part of the execution union of a BackupClass: what says who
// executes the class. It is named Type.FieldOrMethod after its declaration in
// backupAPIPath.
type member string

const (
	executionMode    = member("BackupClassSpec.ExecutionMode")
	providerManaged  = member("BackupClassSpec.ProviderManaged")
	executorProvider = member("BackupClassSpec.ExecutorProvider")
	providerRef      = member("ProviderManagedSpec.ProviderRef")
)

// sanction lets a function use some members of the execution union, for a
// reason.
type sanction struct {
	uses []member
	why  string
}

// executionUnionReaders returns the functions of provider-runtime that may
// use the execution union of a BackupClass, keyed by package and function as
// types.Func.FullName spells them, relative to provider-runtime. Each either
// decides which classes this provider claims and executes, or reads the
// per-mode configuration of a class it executes. None decides whether a class
// suits a provider or a storage: that is the verdict core publishes.
func executionUnionReaders() map[string]sanction {
	return map[string]sanction{
		// The claim path: is this provider the class's executor?
		"(*reconciler.backupClassClaimReconciler).claims": {
			uses: []member{executorProvider},
			why:  "selects the classes this provider claims and judges for Accepted",
		},
		"reconciler.resolveBackupOwnership": {
			uses: []member{executorProvider},
			why:  "executes a Backup only when this provider executes its class, and ignores it otherwise",
		},
		"reconciler.resolveRestoreOwnership": {
			uses: []member{executorProvider},
			why:  "executes a Restore only when this provider executes its read class, and ignores it otherwise",
		},
		"reconciler.resolveImportOwnership": {
			uses: []member{executorProvider},
			why:  "imports through a class only when this provider executes it, and ignores it otherwise",
		},

		// The mode of the class an operation was executed with, recorded on it.
		"(*reconciler.backupRuntimeReconciler).updateStatus": {
			uses: []member{executionMode},
			why:  "records the executed class's mode in Backup.status.executionMode",
		},
		"reconciler.applyBackupExecutionStatus": {
			uses: []member{executionMode},
			why:  "records the executed class's mode in Backup.status.executionMode",
		},
		"(*reconciler.restoreRuntimeReconciler).updateStatus": {
			uses: []member{executionMode},
			why:  "records the executed class's mode in Restore.status.executionMode",
		},
		"reconciler.applyRestoreExecutionStatus": {
			uses: []member{executionMode},
			why:  "records the executed class's mode in Restore.status.executionMode",
		},

		// The per-mode configuration of a class.
		"internal/backupclass.applies": {
			uses: []member{executionMode},
			why:  "limits LimitErrors and PITRErrors to the ProviderManaged member they read; other modes pass",
		},
		"internal/backupclass.LimitErrors": {
			uses: []member{providerManaged},
			why:  "enforces providerManaged.limits on the Instance's backup configuration",
		},
		"internal/backupclass.PITRErrors": {
			uses: []member{providerManaged},
			why:  "validates PITR parameters against providerManaged.pitrParametersSchema",
		},
		"(*controller.Context).BackupClassLimits": {
			uses: []member{providerManaged},
			why:  "hands providerManaged.limits to the provider's own checks",
		},
		"(*controller.Context).PITRParametersSchema": {
			uses: []member{providerManaged},
			why:  "hands providerManaged.pitrParametersSchema to the provider",
		},
		"(*controller.Context).ReconcileDataSource": {
			uses: []member{providerManaged},
			why:  "reads providerManaged.supportsPITR once the published verdict is Compatible",
		},
		"controller.ValidateRestorePITR": {
			uses: []member{executionMode, providerManaged},
			why:  "gates point-in-time Restores on providerManaged.supportsPITR; classes of other modes are not gated",
		},
	}
}

const guidance = `Spec 019 §4.8 invariant 1: outside its claim path, provider-runtime decides whether a BackupClass suits a provider or a storage from the verdicts core publishes, never from the class's execution union, so that a later execution mode is an addition that breaks nothing.
  - To decide compatibility, read the published verdict instead: backupclass.ProviderVerdict (backupv1alpha1.CurrentProviderVerdict) for a provider, backupv1alpha1.CurrentVerdict for a storage.
  - If the function decides which classes this provider claims and executes ("is the executor me?"), or reads the per-mode configuration of a class this provider executes (providerManaged.limits, supportsPITR, pitrParametersSchema), add it to executionUnionReaders in provider-runtime/invariants_test.go with the members it uses and a one-line reason.`

// TestOnlyTheClaimPathReadsTheExecutionMode fails when the non-test code of
// provider-runtime uses the execution union of a BackupClass outside
// executionUnionReaders, or when an entry there sanctions a use no longer
// made.
func TestOnlyTheClaimPathReadsTheExecutionMode(t *testing.T) {
	t.Parallel()

	readers := executionUnionReaders()
	for fn, s := range readers {
		assert.NotEmpty(t, s.uses, "executionUnionReaders[%q] sanctions no member", fn)
		assert.NotEmpty(t, s.why, "executionUnionReaders[%q] gives no reason", fn)
	}

	violations, stale := checkUses(executionUnionUses(t, loadPackages(t, "./...")), readers)
	assert.Empty(t, lines(violations), guidance)
	assert.Empty(t, lines(stale),
		"executionUnionReaders in provider-runtime/invariants_test.go sanctions uses that are no longer made: "+
			"remove them, or follow the function they moved to",
	)
}

// TestExecutionUnionAnalyser proves the analyser reports uses of the
// execution union outside an allowlist through aliases, defined types,
// embedding, method values and expressions, closures and package-level
// variables, ignores fields of the same name on other types, and reports
// stale allowlist entries.
func TestExecutionUnionAnalyser(t *testing.T) {
	t.Parallel()

	uses := executionUnionUses(t, loadPackages(t, "./testdata/executionunion"))
	violations, stale := checkUses(uses, map[string]sanction{
		"testdata/executionunion.Claims": {uses: []member{executorProvider, executionMode}},
		"testdata/executionunion.Limits": {uses: []member{providerManaged}},
		"testdata/executionunion.Gone":   {uses: []member{executionMode}},
	})

	reported := make([]string, 0, len(violations))
	for _, v := range violations {
		reported = append(reported, v.what())
	}
	wantReported := []string{
		"testdata/executionunion.Limits uses BackupClassSpec.ExecutorProvider",
		"testdata/executionunion.Compatible uses BackupClassSpec.ExecutionMode",
		"testdata/executionunion.throughAlias uses BackupClassSpec.ProviderManaged",
		"testdata/executionunion.throughRename uses BackupClassSpec.ExecutionMode",
		"testdata/executionunion.throughEmbedding uses BackupClassSpec.ExecutorProvider",
		"testdata/executionunion.methodValue uses BackupClassSpec.ExecutorProvider",
		"testdata/executionunion.throughIdentity uses ProviderManagedSpec.ProviderRef",
		"testdata/executionunion.inClosure uses BackupClassSpec.ProviderManaged",
		"(*testdata/executionunion.reconciler).owns uses BackupClassSpec.ExecutorProvider",
		"testdata/executionunion.methodExpression uses BackupClassSpec.ExecutorProvider",
	}
	assert.ElementsMatch(t, wantReported, reported)
	wantStale := []string{
		"testdata/executionunion.Claims does not use BackupClassSpec.ExecutionMode",
		"testdata/executionunion.Gone does not use BackupClassSpec.ExecutionMode",
	}
	assert.Equal(t, wantStale, stale)
	for _, v := range violations {
		assert.True(t, strings.HasPrefix(v.pos.Filename, filepath.Join("testdata", "executionunion")+string(filepath.Separator)),
			"position %s is not relative to provider-runtime", v.pos,
		)
	}
}

// use is one use of a member of the execution union, in the function fn.
type use struct {
	fn     string
	member member
	pos    token.Position
}

func (u use) String() string { return fmt.Sprintf("%s: %s", u.pos, u.what()) }

func (u use) what() string { return fmt.Sprintf("%s uses %s", u.fn, u.member) }

// loadPackages loads and type-checks the non-test packages matching patterns,
// relative to provider-runtime.
func loadPackages(t *testing.T, patterns ...string) []*packages.Package {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	require.NoError(t, err)
	require.NotEmpty(t, pkgs, "no package matches %v", patterns)

	// A package that does not type-check has gaps in its type information,
	// where a use could hide.
	var errs []string
	for _, p := range pkgs {
		for _, e := range p.Errors {
			errs = append(errs, e.Error())
		}
	}
	require.Empty(t, errs, "packages matching %v do not load", patterns)
	return pkgs
}

// executionUnionUses returns every use, in pkgs, of a member of the execution
// union, resolved through go/types: the object a selector selects, whatever
// the type it is selected on is called.
func executionUnionUses(t *testing.T, pkgs []*packages.Package) []use {
	t.Helper()

	members := executionUnionObjects(t, pkgs)
	wd, err := os.Getwd()
	require.NoError(t, err)

	var uses []use
	for _, p := range pkgs {
		inspect := func(fn string, node ast.Node) {
			ast.Inspect(node, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				// A qualified identifier (pkg.Name) is no selection.
				if s, ok := p.TypesInfo.Selections[sel]; ok {
					if m, ok := members[s.Obj()]; ok {
						pos := p.Fset.Position(sel.Sel.Pos())
						if rel, err := filepath.Rel(wd, pos.Filename); err == nil {
							pos.Filename = rel
						}
						uses = append(uses, use{fn: fn, member: m, pos: pos})
					}
				}
				return true
			})
		}

		pkg := strings.TrimPrefix(p.PkgPath, runtimePath)
		for _, f := range p.Syntax {
			for _, decl := range f.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					fn, ok := p.TypesInfo.Defs[d.Name].(*types.Func)
					require.True(t, ok, "%s declares no function", p.Fset.Position(d.Pos()))
					inspect(strings.ReplaceAll(fn.FullName(), runtimePath, ""), d)
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.ValueSpec:
							names := make([]string, 0, len(s.Names))
							for _, n := range s.Names {
								names = append(names, n.Name)
							}
							inspect(pkg+"."+strings.Join(names, ","), s)
						case *ast.TypeSpec:
							inspect(pkg+"."+s.Name.Name, s)
						}
					}
				}
			}
		}
	}
	return uses
}

// executionUnionObjects resolves every member of the execution union to its
// object in each copy of backupAPIPath that pkgs import, directly or not.
func executionUnionObjects(t *testing.T, pkgs []*packages.Package) map[types.Object]member {
	t.Helper()

	var apis []*types.Package
	seen := make(map[*types.Package]bool)
	var walk func(*types.Package)
	walk = func(p *types.Package) {
		if seen[p] {
			return
		}
		seen[p] = true
		if p.Path() == backupAPIPath {
			apis = append(apis, p)
		}
		for _, imp := range p.Imports() {
			walk(imp)
		}
	}
	for _, p := range pkgs {
		walk(p.Types)
	}
	require.NotEmpty(t, apis, "no analysed package imports %s", backupAPIPath)

	objects := make(map[types.Object]member)
	for _, api := range apis {
		for _, m := range []member{executionMode, providerManaged, executorProvider, providerRef} {
			typeName, name, _ := strings.Cut(string(m), ".")
			tn, ok := api.Scope().Lookup(typeName).(*types.TypeName)
			require.True(t, ok, "%s declares no type %s: follow the rename in provider-runtime/invariants_test.go", backupAPIPath, typeName)
			obj, _, _ := types.LookupFieldOrMethod(tn.Type(), true, api, name)
			require.NotNil(t, obj, "%s.%s has no field or method %s: follow the rename in provider-runtime/invariants_test.go",
				backupAPIPath, typeName, name,
			)
			objects[obj] = m
		}
	}
	return objects
}

// checkUses returns the uses that sanctions do not allow, and, sorted, the
// sanctioned uses that are not made.
func checkUses(uses []use, sanctions map[string]sanction) ([]use, []string) {
	made := make(map[string]map[member]bool)
	var violations []use
	for _, u := range uses {
		if made[u.fn] == nil {
			made[u.fn] = make(map[member]bool)
		}
		made[u.fn][u.member] = true
		if !slices.Contains(sanctions[u.fn].uses, u.member) {
			violations = append(violations, u)
		}
	}

	var stale []string
	for fn, s := range sanctions {
		for _, m := range s.uses {
			if !made[fn][m] {
				stale = append(stale, fmt.Sprintf("%s does not use %s", fn, m))
			}
		}
	}
	slices.Sort(stale)
	return violations, stale
}

// lines formats items one per line, for a failure message.
func lines[T any](items []T) string {
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n\t%v", item)
	}
	return b.String()
}
