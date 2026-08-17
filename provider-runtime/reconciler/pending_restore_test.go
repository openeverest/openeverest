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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func newRestoreWithState(name, namespace, instanceName string, state backupv1alpha1.RestoreState) *backupv1alpha1.Restore {
	return &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: instanceName},
		},
		Status: backupv1alpha1.RestoreStatus{
			State: state,
		},
	}
}

func TestPendingRestoresForInstance(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
		},
	}

	// datasource restore — always excluded regardless of state
	datasourceRestore := newRestoreWithState("my-db-datasource", "default", "my-db", backupv1alpha1.RestoreStateRunning)
	// user-initiated restores in various states
	pendingRestore := newRestoreWithState("restore-pending", "default", "my-db", backupv1alpha1.RestoreStatePending)
	runningRestore := newRestoreWithState("restore-running", "default", "my-db", backupv1alpha1.RestoreStateRunning)
	newRestore := newRestoreWithState("restore-new", "default", "my-db", "") // not yet started
	succeededRestore := newRestoreWithState("restore-ok", "default", "my-db", backupv1alpha1.RestoreStateSucceeded)
	failedRestore := newRestoreWithState("restore-fail", "default", "my-db", backupv1alpha1.RestoreStateFailed)
	errorRestore := newRestoreWithState("restore-err", "default", "my-db", backupv1alpha1.RestoreStateError)
	// restore for a different instance — never included
	otherRestore := newRestoreWithState("restore-other", "default", "other-db", backupv1alpha1.RestoreStateRunning)

	c := newFakeClient(scheme, instance,
		datasourceRestore, pendingRestore, runningRestore, newRestore,
		succeededRestore, failedRestore, errorRestore, otherRestore,
	)
	inCtx := controller.NewContext(context.Background(), c, instance, "test-provider")

	got, err := inCtx.PendingRestoresForInstance()
	require.NoError(t, err)

	names := make([]string, len(got))
	for i, r := range got {
		names[i] = r.Name
	}
	assert.ElementsMatch(t, []string{"restore-pending", "restore-running", "restore-new"}, names,
		"should include only non-terminal, non-datasource restores")
}

func TestPendingRestoresForInstance_EmptyWhenNone(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
		},
	}

	c := newFakeClient(scheme, instance)
	inCtx := controller.NewContext(context.Background(), c, instance, "test-provider")

	got, err := inCtx.PendingRestoresForInstance()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetPendingRestore_SetAndGet(t *testing.T) {
	t.Parallel()

	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
	}
	inCtx := controller.NewContext(context.Background(), nil, instance, "test-provider")

	assert.Nil(t, inCtx.GetPendingRestore(), "should be nil before set")

	restore := &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-1", Namespace: "default"},
	}
	inCtx.SetPendingRestore(restore)
	assert.Equal(t, restore, inCtx.GetPendingRestore())
}
