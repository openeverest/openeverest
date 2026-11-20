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

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

func TestValidate_ListMonitoringBindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		params  *api.ListMonitoringBindingsParams
		wantErr bool
	}{
		{desc: "no params"},
		{desc: "no destination", params: &api.ListMonitoringBindingsParams{}},
		{desc: "destination name", params: &api.ListMonitoringBindingsParams{Destination: new("pmm-prod")}},
		{
			desc:    "a second selector term smuggled in",
			params:  &api.ListMonitoringBindingsParams{Destination: new("pmm-prod,openeverest.io/instance=db1")},
			wantErr: true,
		},
		{
			desc:    "not a label value",
			params:  &api.ListMonitoringBindingsParams{Destination: new("team/pmm")},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("ListMonitoringBindings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&monitoringv1alpha1.MonitoringBindingList{Items: []monitoringv1alpha1.MonitoringBinding{
					{ObjectMeta: metav1.ObjectMeta{Name: "db1-pmm-0a1b2c3d", Namespace: "ns1"}},
				}},
				nil,
			)

			h := &validateHandler{next: next}
			list, err := h.ListMonitoringBindings(context.Background(), "prod", "ns1", tc.params)

			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidRequest)
				next.AssertNotCalled(t, "ListMonitoringBindings", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			assert.Equal(t, "db1-pmm-0a1b2c3d", list.Items[0].Name)
		})
	}
}
