// everest
// Copyright (C) 2023 Percona LLC
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

package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type mockClient struct {
	ctrlclient.Client

	listFunc func(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error
}

func (m *mockClient) List(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
	return m.listFunc(ctx, list, opts...)
}

func TestListPaginated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockList  func(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error
		wantItems int
		wantErr   string
	}{
		{
			name: "successful multi-page pagination",
			mockList: func(_ context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
				listOpts := &ctrlclient.ListOptions{}
				for _, opt := range opts {
					opt.ApplyToList(listOpts)
				}

				secretList, ok := list.(*corev1.SecretList)
				if !ok {
					return errors.New("expected *corev1.SecretList")
				}

				switch listOpts.Continue {
				case "":
					secretList.Items = []corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "secret-1"}}}
					secretList.Continue = "page2"
				case "page2":
					secretList.Items = []corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "secret-2"}}}
					secretList.Continue = ""
				default:
					return errors.New("unexpected continue token")
				}

				return nil
			},
			wantItems: 2,
		},
		{
			name: "mid-pagination error",
			mockList: func(_ context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
				listOpts := &ctrlclient.ListOptions{}
				for _, opt := range opts {
					opt.ApplyToList(listOpts)
				}

				if listOpts.Continue == "" {
					secretList, ok := list.(*corev1.SecretList)
					if !ok {
						return errors.New("expected *corev1.SecretList")
					}
					secretList.Items = []corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "secret-1"}}}
					secretList.Continue = "page2"
					return nil
				}

				return errors.New("api server timeout")
			},
			wantErr: "api server timeout",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mClient := &mockClient{listFunc: tt.mockList}

			result := &corev1.SecretList{}
			err := listPaginated(context.Background(), mClient, result,
				func() *corev1.SecretList { return &corev1.SecretList{} },
				func(res, page *corev1.SecretList) { res.Items = append(res.Items, page.Items...) },
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Len(t, result.Items, tt.wantItems)
			}
		})
	}
}
