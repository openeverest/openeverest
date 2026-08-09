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

package configmapadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casbin/casbin/v2/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// testRBACModel is a minimal Casbin RBAC model definition matching the
// production model in data/rbac/model.conf.
const testRBACModel = `
[request_definition]
r = sub, res, act, obj

[policy_definition]
p = sub, res, act, obj

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub) && globMatch(r.res, p.res) && globMatch(r.act, p.act) && globMatch(r.obj, p.obj)
`

// mockK8s implements the k8s interface for testing.
type mockK8s struct {
	cm    *corev1.ConfigMap
	err   error
	delay time.Duration
}

func (m *mockK8s) GetConfigMap(ctx context.Context, _ ctrlclient.ObjectKey) (*corev1.ConfigMap, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.cm, m.err
}

func TestLoadPolicy(t *testing.T) {
	t.Parallel()

	validCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"policy.csv": "p, role:admin, database-clusters, *, *",
		},
	}

	nn := types.NamespacedName{Namespace: "test-ns", Name: "test-cm"}
	l := zap.NewNop().Sugar()

	testCases := []struct {
		desc      string
		mock      *mockK8s
		timeout   time.Duration
		wantErr   bool
		errSubstr string
	}{
		{
			desc: "happy path - loads policy within timeout",
			mock: &mockK8s{
				cm: validCM,
			},
			wantErr: false,
		},
		{
			desc: "timeout - API call exceeds timeout",
			mock: &mockK8s{
				cm:    validCM,
				delay: 500 * time.Millisecond,
			},
			timeout:   100 * time.Millisecond,
			wantErr:   true,
			errSubstr: "context deadline exceeded",
		},
		{
			desc: "API error - wraps error with configmap context",
			mock: &mockK8s{
				err: errors.New("connection refused"),
			},
			wantErr:   true,
			errSubstr: "failed to load RBAC policy configmap",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			var opts []Option
			if tc.timeout > 0 {
				opts = append(opts, WithTimeout(tc.timeout))
			}
			adapter := New(l, tc.mock, nn, opts...)

			m, err := model.NewModelFromString(testRBACModel)
			require.NoError(t, err)

			done := make(chan error, 1)
			go func() {
				done <- adapter.LoadPolicy(m)
			}()

			select {
			case err := <-done:
				if tc.wantErr {
					if tc.errSubstr != "" {
						require.ErrorContains(t, err, tc.errSubstr)
					} else {
						require.Error(t, err)
					}
				} else {
					require.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("LoadPolicy did not return within bounded time — possible hang")
			}
		})
	}
}

func TestDefaultTimeout(t *testing.T) {
	t.Parallel()
	l := zap.NewNop().Sugar()
	nn := types.NamespacedName{Namespace: "ns", Name: "cm"}
	adapter := New(l, &mockK8s{}, nn)
	assert.Equal(t, defaultKubeAPITimeout, adapter.timeout)
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()
	l := zap.NewNop().Sugar()
	nn := types.NamespacedName{Namespace: "ns", Name: "cm"}
	custom := 500 * time.Millisecond
	adapter := New(l, &mockK8s{}, nn, WithTimeout(custom))
	assert.Equal(t, custom, adapter.timeout)
}
