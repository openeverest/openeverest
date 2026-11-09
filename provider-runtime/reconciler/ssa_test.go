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

package reconciler

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// coreFieldOwner stands for core's field manager in these tests.
const coreFieldOwner = "openeverest-core"

// crdTypeConverter makes the fake client's server-side apply follow the
// schemas of the named CRD files (conditions merge by type).
func crdTypeConverter(t *testing.T, files ...string) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	models := make(map[string]*spec.Schema)
	for _, f := range files {
		raw, err := bases.CRDs.ReadFile(f)
		require.NoError(t, err)
		crd := &apiextensionsv1.CustomResourceDefinition{}
		require.NoError(t, yaml.Unmarshal(raw, crd))
		for _, v := range crd.Spec.Versions {
			js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
			require.NoError(t, err)
			s := &spec.Schema{}
			require.NoError(t, json.Unmarshal(js, s))
			s.AddExtension("x-kubernetes-group-version-kind", []any{
				map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
			})
			models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
		}
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

// newSSATestClient returns a fake client whose server-side apply follows
// the Instance and BackupClass CRDs and that returns managed fields.
func newSSATestClient(t *testing.T, objs ...client.Object) client.WithWatch { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	scheme := newTestScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(
			crdTypeConverter(t, "core.openeverest.io_instances.yaml", "backup.openeverest.io_backupclasses.yaml"),
			managedfields.NewDeducedTypeConverter(),
		).
		WithReturnManagedFields().
		WithObjects(objs...).
		WithStatusSubresource(&v1alpha1.Instance{}, &backupv1alpha1.BackupClass{}).
		Build()
}

// statusOwners returns the field managers that own fields of obj's status,
// with their operation.
func statusOwners(obj client.Object) []string {
	var owners []string
	for _, mf := range obj.GetManagedFields() {
		if mf.FieldsV1 != nil && strings.Contains(mf.FieldsV1.GetRawString(), `"f:status"`) {
			owners = append(owners, mf.Manager+"/"+string(mf.Operation))
		}
	}
	slices.Sort(owners)
	return owners
}

func TestBackupClassClaim(t *testing.T) {
	t.Parallel()

	shipped := func() backupv1alpha1.BackupClass {
		bc := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
		bc.Annotations = map[string]string{backupv1alpha1.ProviderVersionAnnotation: "1.2.0"}
		return *bc
	}

	tests := []struct {
		name       string
		class      func() *backupv1alpha1.BackupClass
		wantStatus metav1.ConditionStatus
		wantReason string
	}{
		{
			name: "a shipped class",
			class: func() *backupv1alpha1.BackupClass {
				bc := shipped()
				return &bc
			},
			wantStatus: metav1.ConditionTrue,
			wantReason: backupv1alpha1.ReasonAccepted,
		},
		{
			name: "a drifted class",
			class: func() *backupv1alpha1.BackupClass {
				bc := shipped()
				bc.Spec.SupportsImport = true
				return &bc
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: backupv1alpha1.ReasonSpecDrift,
		},
		{
			name: "a class of the next release",
			class: func() *backupv1alpha1.BackupClass {
				bc := shipped()
				bc.Annotations[backupv1alpha1.ProviderVersionAnnotation] = "1.3.0"
				return &bc
			},
			wantStatus: metav1.ConditionUnknown,
			wantReason: backupv1alpha1.ReasonNewerDefinition,
		},
		{
			name: "a class the provider does not ship",
			class: func() *backupv1alpha1.BackupClass {
				bc := shipped()
				bc.Name = "pbm-physical"
				return &bc
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: backupv1alpha1.ReasonUnknownClass,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			class := tc.class()
			class.Generation = 3
			// Core's verdict, which the claim must not touch.
			class.Status.Providers = []backupv1alpha1.BackupClassProviderCompatibility{{
				Name: "test-provider", Compatibility: backupv1alpha1.CompatibilityPending,
				Reason: backupv1alpha1.ReasonClassNotAccepted, ClassGeneration: 3,
			}}
			c := newSSATestClient(t, class)
			r := &backupClassClaimReconciler{client: c, provider: "test-provider", shipped: []backupv1alpha1.BackupClass{shipped()}}

			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(class)})
			require.NoError(t, err)

			got := &backupv1alpha1.BackupClass{}
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(class), got))
			cond := meta.FindStatusCondition(got.Status.Conditions, backupv1alpha1.BackupClassConditionAccepted)
			require.NotNil(t, cond)
			assert.Equal(t, tc.wantStatus, cond.Status)
			assert.Equal(t, tc.wantReason, cond.Reason)
			assert.Equal(t, int64(3), cond.ObservedGeneration)
			assert.Equal(t, class.Status.Providers, got.Status.Providers, "core's field is left alone")
			assert.Contains(t, statusOwners(got), controller.StatusFieldOwner("test-provider")+"/Apply")
		})
	}
}

// TestBackupClassClaim_IgnoresOthers checks that the claim writes nothing on
// classes another executor claims and does not rewrite a current claim.
func TestBackupClassClaim_IgnoresOthers(t *testing.T) {
	t.Parallel()

	foreign := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "other")
	foreign.Name = "other-class"
	otherMode := testBackupClass(otherMode)
	otherMode.Name = "job-class"
	claimed := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
	claimed.Status.Conditions = []metav1.Condition{{
		Type: backupv1alpha1.BackupClassConditionAccepted, Status: metav1.ConditionTrue, Reason: backupv1alpha1.ReasonAccepted,
		Message: "provider test-provider executes BackupClass pbm", ObservedGeneration: 1, LastTransitionTime: metav1.Now(),
	}}
	var writes int
	c := interceptor.NewClient(newSSATestClient(t, foreign, otherMode, claimed), interceptor.Funcs{
		SubResourceApply: func(context.Context, client.Client, string, runtime.ApplyConfiguration, ...client.SubResourceApplyOption) error {
			writes++
			return nil
		},
	})
	r := &backupClassClaimReconciler{
		client:   c,
		provider: "test-provider",
		shipped:  []backupv1alpha1.BackupClass{*testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")},
	}

	for _, bc := range []*backupv1alpha1.BackupClass{foreign, otherMode, claimed} {
		_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(bc)})
		require.NoError(t, err)
	}
	assert.Zero(t, writes)
	assert.False(t, r.claims(foreign))
	assert.False(t, r.claims(otherMode))
	assert.True(t, r.claims(claimed))
}

// applyCoreCondition applies conditions on the Instance's status as core
// would, under its own field manager.
func applyCoreCondition(t *testing.T, c client.Client, in *v1alpha1.Instance, conds ...metav1.Condition) {
	t.Helper()
	list := make([]any, 0, len(conds))
	for i := range conds {
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&conds[i])
		require.NoError(t, err)
		list = append(list, raw)
	}
	require.NoError(t, monitoring.ApplyStatus(t.Context(), c, in, coreFieldOwner, map[string]any{"conditions": list}))
}

// coreCondition is a condition of a type the runtime does not write.
func coreCondition() metav1.Condition {
	return metav1.Condition{
		Type: "BackupJobConfigured", Status: metav1.ConditionTrue, Reason: "Configured",
		Message: "set by core", ObservedGeneration: 2, LastTransitionTime: metav1.Now(),
	}
}

// TestReconcile_KeepsStatusOfOtherWriters is the conformance test of spec
// 019 §4.8 invariant 7: a status field core applies on an Instance survives
// a reconcile, also by a runtime that cannot decode it, and stays core's to
// remove.
func TestReconcile_KeepsStatusOfOtherWriters(t *testing.T) {
	t.Parallel()

	in := readyTestInstance(nil)
	c := newSSATestClient(t, testProviderCR(), in)
	applyCoreCondition(t, c, in, coreCondition())

	// An older runtime: what it decodes lacks the field core wrote.
	older := interceptor.NewClient(c, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if err := c.Get(ctx, key, obj, opts...); err != nil {
				return err
			}
			if got, ok := obj.(*v1alpha1.Instance); ok {
				meta.RemoveStatusCondition(&got.Status.Conditions, coreCondition().Type)
			}
			return nil
		},
	})
	for _, rc := range []client.Client{c, older} {
		r := &ProviderReconciler{provider: &readyTestProvider{status: controller.Ready()}, Client: rc}
		_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(in)})
		require.NoError(t, err)

		got := &v1alpha1.Instance{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(in), got))
		core := meta.FindStatusCondition(got.Status.Conditions, coreCondition().Type)
		require.NotNil(t, core, "the core's condition survives the reconcile")
		assert.Equal(t, "set by core", core.Message)
		ready := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionReady)
		require.NotNil(t, ready)
		assert.Equal(t, metav1.ConditionTrue, ready.Status)
		assert.Equal(t, v1alpha1.InstancePhaseReady, got.Status.Phase)
		wantOwners := []string{coreFieldOwner + "/Apply", controller.StatusFieldOwner("test-provider") + "/Apply"}
		assert.Equal(t, wantOwners, statusOwners(got))
	}

	applyCoreCondition(t, c, in)
	got := &v1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(in), got))
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, coreCondition().Type),
		"the runtime never took the core's condition over, so core removes it",
	)
	assert.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionReady))
}

// TestReconcile_StatusApplyRemovesWhatItStopsSetting checks that a field the
// runtime no longer sets is removed, as Update did.
func TestReconcile_StatusApplyRemovesWhatItStopsSetting(t *testing.T) {
	t.Parallel()

	in := readyTestInstance(nil)
	c := newSSATestClient(t, testProviderCR(), in)
	p := &readyTestProvider{status: controller.Status{Phase: v1alpha1.InstancePhaseProvisioning, Message: "creating pods"}}
	r := &ProviderReconciler{provider: p, Client: c}
	key := client.ObjectKeyFromObject(in)

	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	require.NoError(t, err)
	got := &v1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), key, got))
	assert.Equal(t, "creating pods", got.Status.Message)

	p.status = controller.Ready()
	_, err = r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	require.NoError(t, err)
	require.NoError(t, c.Get(t.Context(), key, got))
	assert.Equal(t, v1alpha1.InstancePhaseReady, got.Status.Phase)
	assert.Empty(t, got.Status.Message)
}

func TestDefaultFieldManager(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "manager", defaultFieldManager("manager/v0.0.0 (linux/amd64) kubernetes/$Format"))
	assert.NotEmpty(t, defaultFieldManager(""))
	assert.Len(t, defaultFieldManager(strings.Repeat("a", 200)+"/v1"), maxFieldManagerLength)
}

// TestRuntimeConditionCoversEverySetCondition fails when the runtime sets an
// Instance condition whose type runtimeCondition does not list: the status
// apply would drop it.
func TestRuntimeConditionCoversEverySetCondition(t *testing.T) {
	t.Parallel()
	values := stringConstants(t, filepath.Join("..", "..", "api", "core", "v1alpha1"))

	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	var calls int
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if fn, ok := call.Fun.(*ast.Ident); !ok || fn.Name != "setCondition" || len(call.Args) < 2 {
				return true
			}
			calls++
			sel, ok := call.Args[1].(*ast.SelectorExpr)
			if !assert.True(t, ok, "%s: setCondition takes a v1alpha1 condition constant", fset.Position(call.Pos())) {
				return true
			}
			value, ok := values[sel.Sel.Name]
			if assert.True(t, ok, "%s: %s is not a constant of api/core/v1alpha1", fset.Position(call.Pos()), sel.Sel.Name) {
				assert.True(t, runtimeCondition(value), "%s: add v1alpha1.%s to runtimeCondition", fset.Position(call.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
	assert.NotZero(t, calls)
}

// stringConstants returns the string constants declared in the package in
// dir, by name.
func stringConstants(t *testing.T, dir string) map[string]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	out := make(map[string]string)
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, s := range gen.Specs {
				vs, ok := s.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, id := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						v, err := strconv.Unquote(lit.Value)
						require.NoError(t, err)
						out[id.Name] = v
					}
				}
			}
		}
	}
	return out
}
