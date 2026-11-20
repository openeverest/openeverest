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

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestValidate_CreateInstance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc        string
		annotations map[string]string
		wantErr     error
	}{
		{
			desc: "no annotations",
		},
		{
			desc: "only instance preset",
			annotations: map[string]string{
				common.InstancePresetAnnotation: "standard-mysql",
			},
		},
		{
			desc: "only cluster instance preset",
			annotations: map[string]string{
				common.ClusterInstancePresetAnnotation: "cluster-mysql",
			},
		},
		{
			desc: "both annotations empty",
			annotations: map[string]string{
				common.InstancePresetAnnotation:        "",
				common.ClusterInstancePresetAnnotation: "",
			},
		},
		{
			desc: "cluster preset set, instance preset empty",
			annotations: map[string]string{
				common.InstancePresetAnnotation:        "",
				common.ClusterInstancePresetAnnotation: "cluster-mysql",
			},
		},
		{
			desc: "both presets set",
			annotations: map[string]string{
				common.InstancePresetAnnotation:        "standard-mysql",
				common.ClusterInstancePresetAnnotation: "cluster-mysql",
			},
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			instance := &corev1alpha1.Instance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "db1",
					Namespace:   "ns1",
					Annotations: tc.annotations,
				},
			}

			next := &handlers.MockHandler{}
			next.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(instance, nil)

			h := &validateHandler{next: next}

			result, err := h.CreateInstance(context.Background(), "prod", instance)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, "db1", result.Name)
			next.AssertCalled(t, "CreateInstance", mock.Anything, "prod", instance)
		})
	}
}

func TestValidate_PatchInstance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		patch   string
		wantErr bool
		wantMsg string
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
		{
			desc:    "userSecretRef",
			patch:   `{"spec":{"userSecretRef":{"name":"other-secret"}}}`,
			wantErr: true,
		},
		{
			desc:  "monitoring destinations",
			patch: `{"spec":{"monitoring":{"destinations":[{"name":"pmm","destinationRef":{"name":"pmm-prod"}}]}}}`,
		},
		{
			desc:    "spec not an object",
			patch:   `{"spec":"x"}`,
			wantErr: true,
			wantMsg: "spec: unexpected JSON string",
		},
		{
			desc:    "monitoring destinations not a list",
			patch:   `{"spec":{"monitoring":{"destinations":{}}}}`,
			wantErr: true,
			wantMsg: "spec.monitoring.destinations: unexpected JSON object",
		},
		{
			desc:    "monitoring destination reference not a string",
			patch:   `{"spec":{"monitoring":{"destinations":[{"name":"pmm","destinationRef":{"name":1}}]}}}`,
			wantErr: true,
			wantMsg: "spec.monitoring.destinations.0.destinationRef.name: unexpected JSON number",
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
				if tc.wantMsg != "" {
					require.EqualError(t, err, ErrInvalidRequest.Error()+"\n"+tc.wantMsg)
				}
				next.AssertNotCalled(t, "PatchInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "db1", result.Name)
		})
	}
}

func TestValidate_UpdateInstance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		current *commonv1alpha1.SecretRef
		update  *commonv1alpha1.SecretRef
		wantErr bool
	}{
		{
			desc:    "no userSecretRef",
			current: nil,
			update:  nil,
		},
		{
			desc:    "unchanged userSecretRef",
			current: &commonv1alpha1.SecretRef{Name: "my-secret"},
			update:  &commonv1alpha1.SecretRef{Name: "my-secret"},
		},
		{
			desc:    "userSecretRef on update",
			current: nil,
			update:  &commonv1alpha1.SecretRef{Name: "my-secret"},
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("GetInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
					Spec:       corev1alpha1.InstanceSpec{UserSecretRef: tc.current},
				},
				nil,
			)
			next.On("UpdateInstance", mock.Anything, mock.Anything, mock.Anything).Return(
				&corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
					Spec:       corev1alpha1.InstanceSpec{UserSecretRef: tc.update},
				},
				nil,
			)

			h := &validateHandler{next: next}
			instance := &corev1alpha1.Instance{
				ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				Spec:       corev1alpha1.InstanceSpec{UserSecretRef: tc.update},
			}
			result, err := h.UpdateInstance(ctx, "prod", instance)

			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidRequest)
				next.AssertNotCalled(t, "UpdateInstance", mock.Anything, mock.Anything, mock.Anything)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "db1", result.Name)
		})
	}
}
