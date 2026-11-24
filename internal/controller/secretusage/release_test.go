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

package secretusage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func storedSecret(t *testing.T, c client.Client, name string) *corev1.Secret {
	t.Helper()
	secret := &corev1.Secret{}
	err := c.Get(t.Context(), types.NamespacedName{Namespace: testNamespace, Name: name}, secret)
	if apierrors.IsNotFound(err) {
		return nil
	}
	require.NoError(t, err)
	return secret
}

func TestRelease(t *testing.T) {
	t.Parallel()

	held := marked(credentials("pmm-token"),
		map[string]string{destinationKey: "1", podsCountKey: "2", "example.com/note": "x"},
		"example.com/keep", finalizer,
	)
	// Labels removed while core was down: still released.
	unlabelled := marked(newSecret("unlabelled", nil), counts("1"), finalizer)
	// Counted but unused: a count left behind would no longer be kept up to date.
	counted := marked(caBundle("corp-ca"), counts("0"))
	deleting := deleted(marked(caBundle("deleting-ca"), counts("1"), finalizer))
	unmarked := marked(newSecret("someone-elses", nil), map[string]string{"example.com/note": "x"}, "example.com/keep")
	c := newClient(t, interceptor.Funcs{}, held, unlabelled, counted, deleting, unmarked,
		destination("pmm-prod", "pmm-token", ""),
	)
	untouched := storedSecret(t, c, "someone-elses").ResourceVersion

	released, err := Release(t.Context(), c, []Referrer{destinations(), pods()})

	require.NoError(t, err)
	assert.Equal(t, 4, released)
	stored := storedSecret(t, c, "pmm-token")
	assert.Equal(t, []string{"example.com/keep"}, stored.Finalizers)
	assert.Equal(t, map[string]string{"example.com/note": "x"}, stored.Annotations)
	assert.Equal(t, held.Labels, stored.Labels)
	assert.Equal(t, held.Data, stored.Data)
	assert.Empty(t, storedSecret(t, c, "unlabelled").Finalizers)
	assert.Empty(t, storedSecret(t, c, "unlabelled").Annotations)
	assert.Empty(t, storedSecret(t, c, "corp-ca").Annotations)
	assert.Nil(t, storedSecret(t, c, "deleting-ca"), "a Secret being deleted goes once released")
	assert.Equal(t, untouched, storedSecret(t, c, "someone-elses").ResourceVersion)

	again, err := Release(t.Context(), c, []Referrer{destinations(), pods()})
	require.NoError(t, err)
	assert.Zero(t, again, "a second run changes nothing")
}

func TestReleaseRetriesConflicts(t *testing.T) {
	t.Parallel()
	patches := 0
	funcs := interceptor.Funcs{
		Patch: func(ctx context.Context, cl client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			patches++
			if patches == 1 {
				// Someone else adds a finalizer between the list and the patch.
				current := &corev1.Secret{}
				require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(obj), current))
				base := current.DeepCopy()
				current.Finalizers = append(current.Finalizers, "example.com/concurrent")
				require.NoError(t, cl.Patch(ctx, current, client.MergeFrom(base)))
			}
			return cl.Patch(ctx, obj, patch, opts...)
		},
	}
	c := newClient(t, funcs, marked(credentials("pmm-token"), counts("1"), finalizer))

	released, err := Release(t.Context(), c, []Referrer{destinations()})

	require.NoError(t, err)
	assert.Equal(t, 1, released)
	assert.Equal(t, 2, patches)
	stored := storedSecret(t, c, "pmm-token")
	assert.Equal(t, []string{"example.com/concurrent"}, stored.Finalizers)
	assert.Empty(t, stored.Annotations)
}

func TestReleaseFollowsListPages(t *testing.T) {
	t.Parallel()
	var continues []string
	funcs := interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			listOpts := &client.ListOptions{}
			listOpts.ApplyOptions(opts)
			continues = append(continues, listOpts.Continue)
			assert.Equal(t, int64(listPageSize), listOpts.Limit)
			if err := cl.List(ctx, list, opts...); err != nil {
				return err
			}
			// One Secret per page: the first, then the rest.
			partial, ok := list.(*metav1.PartialObjectMetadataList)
			require.True(t, ok, "Secrets are listed metadata only")
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
	c := newClient(t, funcs, marked(credentials("a"), counts("1"), finalizer), marked(credentials("b"), counts("1"), finalizer))

	released, err := Release(t.Context(), c, []Referrer{destinations()})

	require.NoError(t, err)
	assert.Equal(t, 2, released)
	assert.Equal(t, []string{"", "page-2"}, continues)
	assert.Empty(t, storedSecret(t, c, "a").Finalizers)
	assert.Empty(t, storedSecret(t, c, "b").Finalizers)
}

func TestReleaseReportsFailuresAndGoesOn(t *testing.T) {
	t.Parallel()
	c := newClient(t, interceptor.Funcs{
		Patch: func(ctx context.Context, cl client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			switch obj.GetName() {
			case "forbidden":
				return apierrors.NewForbidden(corev1.Resource("secrets"), obj.GetName(), errors.New("no"))
			case "gone":
				return apierrors.NewNotFound(corev1.Resource("secrets"), obj.GetName())
			}
			return cl.Patch(ctx, obj, patch, opts...)
		},
	},
		marked(credentials("forbidden"), counts("1"), finalizer),
		marked(credentials("gone"), counts("1"), finalizer),
		marked(credentials("pmm-token"), counts("1"), finalizer),
	)

	released, err := Release(t.Context(), c, []Referrer{destinations()})

	require.ErrorContains(t, err, "forbidden")
	assert.True(t, apierrors.IsForbidden(err))
	assert.Equal(t, 1, released, "the other Secrets are still released; one gone needs nothing")
	assert.Empty(t, storedSecret(t, c, "pmm-token").Finalizers)
	assert.Equal(t, []string{finalizer}, storedSecret(t, c, "forbidden").Finalizers)
}

func TestReleaseReportsAListError(t *testing.T) {
	t.Parallel()
	boom := errors.New("connection refused")
	c := newClient(t, interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error { return boom },
	})

	_, err := Release(t.Context(), c, []Referrer{destinations()})

	require.ErrorIs(t, err, boom)
}
