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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// referenceTestProvider adds ValidateReferences to readyTestProvider.
type referenceTestProvider struct {
	readyTestProvider

	referenceErrs field.ErrorList
	referencesErr error
	syncCalls     int
}

func (p *referenceTestProvider) ValidateReferences(*controller.Context) (field.ErrorList, error) {
	return p.referenceErrs, p.referencesErr
}

func (p *referenceTestProvider) Sync(*controller.Context) error {
	p.syncCalls++
	return nil
}

func TestReconcile_DeclaredChecksRunBeforeValidate(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	provider := &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec:       v1alpha1.ProviderSpec{Versions: []v1alpha1.VersionBundle{{Name: "7.0"}}},
	}
	in := readyTestInstance(func(in *v1alpha1.Instance) { in.Spec.Version = "9.0" })
	r := newReadyTestReconciler(p, provider, in)

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)
	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonValidationFailed)
	assert.Equal(t, `spec.version: Unsupported value: "9.0": supported values: "7.0"`, c.Message)
	assert.Zero(t, p.calls, "Validate runs only once the declared checks pass")
}

func TestReconcile_InvalidReferenceIsCheckedAgain(t *testing.T) {
	t.Parallel()
	p := &referenceTestProvider{referenceErrs: field.ErrorList{field.Invalid(
		field.NewPath("spec", "userSecretRef", "name"), "orders-users", `missing "MONGODB_BACKUP_USER"`,
	)}}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	r := newReadyTestReconciler(p, provider, readyTestInstance(nil))
	key := client.ObjectKey{Namespace: "team-a", Name: "orders"}

	res, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.Equal(t, referenceRecheck, res.RequeueAfter, "a referent's contents can change without an event on the Instance")
	got := &v1alpha1.Instance{}
	require.NoError(t, r.Get(t.Context(), key, got))
	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonInvalidReference)
	assert.Equal(t, `spec.userSecretRef.name: Invalid value: "orders-users": missing "MONGODB_BACKUP_USER"`, c.Message)

	p.referenceErrs = nil
	p.status = controller.Ready()
	got, err = reconcileAndGet(t, r)
	require.NoError(t, err)
	requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
}

func TestReconcile_ReferenceCheckThatCouldNotRunIsRetried(t *testing.T) {
	t.Parallel()
	p := &referenceTestProvider{referencesErr: errors.New("connection refused")}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	r := newReadyTestReconciler(p, provider, readyTestInstance(nil))

	_, err := reconcileAndGet(t, r)
	require.ErrorContains(t, err, "connection refused")
	assert.Zero(t, p.syncCalls, "Sync waits for the check")
}

func TestReconcile_WarningsAreRecordedOncePerGeneration(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	provider := &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec: v1alpha1.ProviderSpec{
			Components: map[string]v1alpha1.Component{"engine": {Type: "mongod"}},
			ComponentTypes: map[string]v1alpha1.ComponentType{"mongod": {
				Versions: []v1alpha1.ComponentVersion{{Version: "6.0.1", Deprecated: true}},
			}},
		},
	}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.UID = "orders-uid"
		in.Spec.Components = map[string]v1alpha1.ComponentSpec{"engine": {Version: "6.0.1"}}
	})
	r := newReadyTestReconciler(p, provider, in)
	recorder := events.NewFakeRecorder(10)
	r.recorder = recorder

	for range 2 {
		_, err := reconcileAndGet(t, r)
		require.NoError(t, err)
	}
	require.Len(t, recorder.Events, 1)
	assert.Equal(t,
		"Warning ValidationWarning spec.components[engine].version: version 6.0.1 is deprecated",
		<-recorder.Events,
	)

	in.Generation++
	r.recordWarnings(in, []string{"again"})
	require.Len(t, recorder.Events, 1, "a new generation records its warnings again")
}
