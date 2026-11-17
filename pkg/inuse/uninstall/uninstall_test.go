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

package uninstall

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	storageFinalizer = "backupstorage.backup.openeverest.io/in-use-protection"
	otherFinalizer   = "example.com/keep"
)

func newClient(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) client.WithWatch { //nolint:ireturn
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).WithInterceptorFuncs(funcs).Build()
}

func objectMeta(namespace, name string, finalizers ...string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: namespace, Name: name, Finalizers: finalizers}
}

func destination(namespace, name string, finalizers ...string) *monitoringv1alpha1.MonitoringDestination {
	return &monitoringv1alpha1.MonitoringDestination{ObjectMeta: objectMeta(namespace, name, finalizers...)}
}

func storage(namespace, name string, finalizers ...string) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{ObjectMeta: objectMeta(namespace, name, finalizers...)}
}

func secret(namespace, name string, finalizers ...string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: objectMeta(namespace, name, finalizers...)}
}

func deleting[T client.Object](obj T) T {
	obj.SetDeletionTimestamp(&metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	return obj
}

func kindOf(obj runtime.Object) string {
	return obj.GetObjectKind().GroupVersionKind().Kind
}

func heldError(t *testing.T, err error) *HeldError {
	t.Helper()
	held, ok := errors.AsType[*HeldError](err)
	require.True(t, ok, "want a *HeldError, got %v", err)
	return held
}

func TestCheckPassesWhenNothingIsHeld(t *testing.T) {
	t.Parallel()

	c := newClient(t, interceptor.Funcs{},
		destination("team-a", "prometheus"),
		storage("team-a", "s3", otherFinalizer),
		secret("team-b", "credentials", otherFinalizer),
		// Someone else's finalizer with core's name on another kind.
		secret("team-a", "odd", monitoringv1alpha1.InUseProtectionFinalizer),
	)

	require.NoError(t, Check(t.Context(), c))
}

func TestCheckNamesWhatIsHeld(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		obj  client.Object
		want Held
	}{
		{
			name: "MonitoringDestination",
			obj:  destination("team-a", "prometheus", otherFinalizer, monitoringv1alpha1.InUseProtectionFinalizer),
			want: Held{Kind: "MonitoringDestination", Objects: []Object{{Namespace: "team-a", Name: "prometheus"}}},
		},
		{
			name: "BackupStorage",
			obj:  storage("team-b", "minio", storageFinalizer),
			want: Held{Kind: "BackupStorage", Objects: []Object{{Namespace: "team-b", Name: "minio"}}},
		},
		{
			name: "Secret",
			obj:  deleting(secret("team-a", "credentials", common.OpenEverestInUseProtectionFinalizer)),
			want: Held{Kind: "Secret", Objects: []Object{{Namespace: "team-a", Name: "credentials", Deleting: true}}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newClient(t, interceptor.Funcs{}, tc.obj)

			err := Check(t.Context(), c)

			held := heldError(t, err)
			assert.Equal(t, []Held{tc.want}, held.Held)
			assert.Contains(t, err.Error(), "1 "+tc.name+": "+tc.want.Objects[0].String())
		})
	}
}

func TestCheckMessage(t *testing.T) {
	t.Parallel()

	c := newClient(t, interceptor.Funcs{},
		destination("team-b", "pmm", monitoringv1alpha1.InUseProtectionFinalizer),
		deleting(destination("team-a", "prometheus", monitoringv1alpha1.InUseProtectionFinalizer)),
		storage("team-a", "s3", storageFinalizer),
	)

	err := Check(t.Context(), c)

	require.Error(t, err)
	assert.Equal(t, "OpenEverest still protects 3 objects with finalizers that only its controller removes; "+
		"uninstalling it now would leave them, and the namespaces holding them, impossible to delete:\n"+
		"  2 MonitoringDestinations: team-a/prometheus (being deleted), team-b/pmm\n"+
		"  1 BackupStorage: team-a/s3\n"+
		"Delete these objects, or the namespaces holding them, while OpenEverest is still running, "+
		"so that it releases them, then uninstall again. `everestctl uninstall` does this for the database namespaces. "+
		"An object being deleted is released once nothing uses it any more.\n"+
		"To uninstall anyway, run `helm uninstall --no-hooks`, then remove the finalizers by hand: "+DocsURL,
		err.Error(),
	)
	assert.True(t, heldError(t, err).AnyDeleting())
}

func TestCheckCapsTheNames(t *testing.T) {
	t.Parallel()

	objs := make([]client.Object, 0, maxNamed+3)
	for i := range maxNamed + 3 {
		objs = append(objs, secret("team-a", fmt.Sprintf("s%02d", i), common.OpenEverestInUseProtectionFinalizer))
	}
	c := newClient(t, interceptor.Funcs{}, objs...)

	err := Check(t.Context(), c)

	held := heldError(t, err)
	require.Len(t, held.Held, 1)
	assert.Len(t, held.Held[0].Objects, maxNamed+3, "the error keeps every object")
	assert.Contains(t, err.Error(), "  13 Secrets: team-a/s00, team-a/s01, team-a/s02, team-a/s03, team-a/s04, "+
		"team-a/s05, team-a/s06, team-a/s07, team-a/s08, team-a/s09 and 3 more\n",
	)
	assert.NotContains(t, err.Error(), "team-a/s10")
}

func TestCheckSkipsKindsNotServed(t *testing.T) {
	t.Parallel()

	funcs := interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if kindOf(list) == "BackupStorageList" {
				return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "backup.openeverest.io", Kind: "BackupStorage"}}
			}
			return cl.List(ctx, list, opts...)
		},
	}
	c := newClient(t, funcs, destination("team-a", "prometheus", monitoringv1alpha1.InUseProtectionFinalizer))

	err := Check(t.Context(), c)

	held := heldError(t, err)
	assert.Equal(t, []Held{{Kind: "MonitoringDestination", Objects: []Object{{Namespace: "team-a", Name: "prometheus"}}}}, held.Held)
}

func TestCheckReportsListErrors(t *testing.T) {
	t.Parallel()

	funcs := interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if kindOf(list) == "SecretList" {
				return apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "", errors.New("no"))
			}
			return cl.List(ctx, list, opts...)
		},
	}
	c := newClient(t, funcs)

	err := Check(t.Context(), c)

	require.Error(t, err)
	_, held := errors.AsType[*HeldError](err)
	assert.False(t, held)
	assert.True(t, apierrors.IsForbidden(err))
}

func TestCheckFollowsListPages(t *testing.T) {
	t.Parallel()

	var continues []string
	funcs := interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if kindOf(list) != "SecretList" {
				return cl.List(ctx, list, opts...)
			}
			listOpts := &client.ListOptions{}
			listOpts.ApplyOptions(opts)
			continues = append(continues, listOpts.Continue)
			if err := cl.List(ctx, list, opts...); err != nil {
				return err
			}
			// One Secret per page.
			partial, ok := list.(*metav1.PartialObjectMetadataList)
			require.True(t, ok)
			require.Len(t, partial.Items, 2)
			if listOpts.Continue == "" {
				partial.Items = partial.Items[:1]
				partial.Continue = "page-2"
			} else {
				partial.Items = partial.Items[1:]
			}
			return nil
		},
	}
	c := newClient(t, funcs,
		secret("team-a", "a", common.OpenEverestInUseProtectionFinalizer),
		secret("team-a", "b", common.OpenEverestInUseProtectionFinalizer),
	)

	err := Check(t.Context(), c)

	assert.Equal(t, []string{"", "page-2"}, continues)
	want := []Held{{Kind: "Secret", Objects: []Object{{Namespace: "team-a", Name: "a"}, {Namespace: "team-a", Name: "b"}}}}
	assert.Equal(t, want, heldError(t, err).Held)
}

func TestCheckNeverWrites(t *testing.T) {
	t.Parallel()

	writes := 0
	write := func() error {
		writes++
		return errors.New("Check must not write")
	}
	funcs := interceptor.Funcs{
		Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return write() },
		Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error { return write() },
		Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
			return write()
		},
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error { return write() },
	}
	c := newClient(t, funcs,
		deleting(destination("team-a", "prometheus", monitoringv1alpha1.InUseProtectionFinalizer)),
		storage("team-a", "s3", storageFinalizer),
	)

	require.Error(t, Check(t.Context(), c))
	assert.Zero(t, writes)
}

func TestHeldErrorAnyDeleting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		held []Held
		want bool
	}{
		{
			name: "one object being deleted",
			held: []Held{
				{Kind: "MonitoringDestination", Objects: []Object{{Name: "a", Deleting: true}}},
				{Kind: "Secret", Objects: []Object{{Name: "b"}}},
			},
			want: true,
		},
		{
			name: "no object being deleted",
			held: []Held{
				{Kind: "MonitoringDestination", Objects: []Object{{Name: "a"}}},
				{Kind: "Secret", Objects: []Object{{Name: "b"}, {Name: "c"}}},
			},
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, (&HeldError{Held: tc.held}).AnyDeleting())
		})
	}
}
