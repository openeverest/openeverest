// everest
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestWithDryRun(t *testing.T) {
	t.Parallel()

	existing := func() *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "cm"},
			Data:       map[string]string{"k": "v"},
		}
	}

	writes := map[string]func(ctx context.Context, c ctrlclient.Client) error{
		"create": func(ctx context.Context, c ctrlclient.Client) error {
			return c.Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "new"}})
		},
		"update": func(ctx context.Context, c ctrlclient.Client) error {
			cm := existing()
			cm.Data["k"] = "changed"
			return c.Update(ctx, cm)
		},
		"patch": func(ctx context.Context, c ctrlclient.Client) error {
			return c.Patch(ctx, existing(), ctrlclient.RawPatch(types.MergePatchType, []byte(`{"data":{"k":"changed"}}`)))
		},
		"delete": func(ctx context.Context, c ctrlclient.Client) error {
			return c.Delete(ctx, existing())
		},
	}

	for name, write := range writes {
		for _, dryRun := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/dryRun=%t", name, dryRun), func(t *testing.T) {
				t.Parallel()

				c := withDryRun(fake.NewClientBuilder().WithObjects(existing()).Build())
				ctx := context.Background()
				if dryRun {
					ctx = WithDryRun(ctx)
				}
				before := &corev1.ConfigMapList{}
				require.NoError(t, c.List(ctx, before))

				require.NoError(t, write(ctx, c))

				after := &corev1.ConfigMapList{}
				require.NoError(t, c.List(ctx, after))
				if dryRun {
					assert.Equal(t, before.Items, after.Items)
				} else {
					assert.NotEqual(t, before.Items, after.Items)
				}
			})
		}
	}
}

func TestWithDryRun_SubResource(t *testing.T) {
	t.Parallel()

	var got ctrlclient.SubResourceUpdateOptions
	recorder := interceptor.NewClient(fake.NewClientBuilder().Build(), interceptor.Funcs{
		SubResourceUpdate: func(_ context.Context, _ ctrlclient.Client, _ string, _ ctrlclient.Object, opts ...ctrlclient.SubResourceUpdateOption) error {
			got.ApplyOptions(opts)
			return nil
		},
	})

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod"}}
	require.NoError(t, withDryRun(recorder).Status().Update(WithDryRun(context.Background()), pod))
	assert.Equal(t, []string{metav1.DryRunAll}, got.DryRun)
}
