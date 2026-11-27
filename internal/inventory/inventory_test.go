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

package inventory

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// TestCollectListsKinds keeps Kinds and Collect in step: /metrics checks the
// informers of Kinds before collecting, so a kind Collect lists but Kinds
// misses could make a scrape wait for the API server.
func TestCollectListsKinds(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	utilruntime.Must(corev1alpha1.AddToScheme(s))
	utilruntime.Must(extensionsv1alpha1.AddToScheme(s))
	utilruntime.Must(backupv1alpha1.AddToScheme(s))
	utilruntime.Must(monitoringv1alpha1.AddToScheme(s))

	var listed []string
	reader := fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			gvk, err := apiutil.GVKForObject(list, s)
			require.NoError(t, err)
			listed = append(listed, strings.TrimSuffix(gvk.Kind, "List"))
			return c.List(ctx, list, opts...)
		},
	}).Build()
	_, err := Collect(context.Background(), reader)
	require.NoError(t, err)

	kinds := make([]string, 0, len(Kinds()))
	for _, obj := range Kinds() {
		gvk, err := apiutil.GVKForObject(obj, s)
		require.NoError(t, err)
		kinds = append(kinds, gvk.Kind)
	}
	assert.ElementsMatch(t, kinds, listed)
}
