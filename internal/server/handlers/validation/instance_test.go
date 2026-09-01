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

package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

func TestValidate_PatchInstance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		patch   string
		wantErr bool
	}{
		{
			desc:  "spec only",
			patch: `{"spec":{"components":{"engine":{"replicas":5}}}}`,
		},
		{
			desc:  "null unsets a spec member",
			patch: `{"spec":{"components":{"proxy":{"parameters":null}}}}`,
		},
		{
			desc:  "conditional patch carrying resourceVersion",
			patch: `{"metadata":{"resourceVersion":"42"},"spec":{"version":"8.0"}}`,
		},
		{
			desc:  "labels are not on the list",
			patch: `{"metadata":{"labels":{"team":"db"}}}`,
		},
		{
			desc:  "a spec member merely named like a forbidden one is fine",
			patch: `{"spec":{"finalizers":"x"}}`,
		},
		{
			desc:    "status",
			patch:   `{"status":{"phase":"Ready"}}`,
			wantErr: true,
		},
		{
			desc:    "ownerReferences",
			patch:   `{"metadata":{"ownerReferences":[]}}`,
			wantErr: true,
		},
		{
			desc:    "finalizers",
			patch:   `{"metadata":{"finalizers":[]}}`,
			wantErr: true,
		},
		{
			desc:    "name",
			patch:   `{"metadata":{"name":"other"}}`,
			wantErr: true,
		},
		{
			desc:    "namespace",
			patch:   `{"metadata":{"namespace":"other"}}`,
			wantErr: true,
		},
		{
			// null deletes the member under RFC 7386, so naming it still counts.
			desc:    "forbidden member set to null",
			patch:   `{"metadata":{"finalizers":null}}`,
			wantErr: true,
		},
		{
			// The typed verbs overwrite these after decoding; a relayed patch
			// would let a caller author its own audit trail.
			desc:    "forged actor annotation",
			patch:   `{"metadata":{"annotations":{"openeverest.io/last-actor-id":"eve"}}}`,
			wantErr: true,
		},
		{
			desc:    "annotations removed wholesale",
			patch:   `{"metadata":{"annotations":null}}`,
			wantErr: true,
		},
		{
			desc:  "annotations may still be set",
			patch: `{"metadata":{"annotations":{"team.example.com/owner":"platform"}}}`,
		},
		{
			// The route the wholesale-removal error points the caller at.
			desc:  "a single annotation may be removed by key",
			patch: `{"metadata":{"annotations":{"team.example.com/owner":null}}}`,
		},
		{
			desc:    "not an object",
			patch:   `[1,2,3]`,
			wantErr: true,
		},
		{
			desc:    "literal null",
			patch:   `null`,
			wantErr: true,
		},
		{
			desc:    "malformed json",
			patch:   `{"spec":`,
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("PatchInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
				nil,
			)

			h := &validateHandler{next: next}
			result, err := h.PatchInstance(ctx, "prod", "ns1", "db1", []byte(tc.patch))

			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidRequest)
				next.AssertNotCalled(t, "PatchInstance")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "db1", result.Name)
		})
	}
}
