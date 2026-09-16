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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

// The member list itself is pinned by TestValidate_PatchInstance, which shares
// the check; this covers the backup storage wiring.
func TestValidate_PatchBackupStorage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		patch   string
		wantErr bool
	}{
		{
			desc:  "spec only",
			patch: `{"spec":{"s3":{"bucket":"bucket-2"}}}`,
		},
		{
			desc:    "status",
			patch:   `{"status":{}}`,
			wantErr: true,
		},
		{
			desc:    "name",
			patch:   `{"metadata":{"name":"other"}}`,
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("PatchBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "bs1", Namespace: "ns1"}},
				nil,
			)

			h := &validateHandler{next: next}
			result, err := h.PatchBackupStorage(ctx, "prod", "ns1", "bs1", []byte(tc.patch))

			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidRequest)
				next.AssertNotCalled(t, "PatchBackupStorage")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "bs1", result.Name)
		})
	}
}
