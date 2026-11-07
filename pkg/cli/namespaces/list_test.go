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

package namespaces

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	olmversion "github.com/operator-framework/api/pkg/lib/version"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestNamespaceLister_Render_JSON(t *testing.T) {
	t.Parallel()

	nsList := []NamespaceInfo{
		{
			Name:               "default",
			InstalledOperators: []string{},
		},
		{
			Name:               "everest",
			InstalledOperators: []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"},
		},
	}

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: false},
	}

	var buf bytes.Buffer
	lister.Render(&buf, nsList)

	assert.JSONEq(t, `[{"name":"default","installedOperators":[]},{"name":"everest","installedOperators":["psmdb(v1.16.0)","postgresql(v1.16.0)"]}]`, buf.String())

	// Verify unmarshaling into struct slice
	var decoded []NamespaceInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	assert.Equal(t, "default", decoded[0].Name)
	assert.Empty(t, decoded[0].InstalledOperators)
	assert.Equal(t, "everest", decoded[1].Name)
	assert.Equal(t, []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"}, decoded[1].InstalledOperators)
}

func TestNamespaceLister_Render_JSON_Empty(t *testing.T) {
	t.Parallel()

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: false},
	}

	// Empty slice should output "[]\n"
	var buf bytes.Buffer
	lister.Render(&buf, []NamespaceInfo{})
	assert.Equal(t, "[]\n", buf.String())

	var decoded []NamespaceInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)

	// nil slice should also output "[]\n", not "null\n"
	buf.Reset()
	lister.Render(&buf, nil)
	assert.Equal(t, "[]\n", buf.String())
}

func TestNamespaceLister_Render_Table(t *testing.T) {
	t.Parallel()

	nsList := []NamespaceInfo{
		{
			Name:               "default",
			InstalledOperators: []string{},
		},
		{
			Name:               "everest",
			InstalledOperators: []string{"psmdb(v1.16.0)", "postgresql(v1.16.0)"},
		},
	}

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, nsList)

	out := buf.String()
	// Headers
	assert.Contains(t, out, "NAMESPACE")
	assert.Contains(t, out, "MANAGED")
	assert.Contains(t, out, "OPERATORS")

	// Rows
	assert.Contains(t, out, "default")
	assert.Contains(t, out, "false")
	assert.Contains(t, out, "everest")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "psmdb(v1.16.0), postgresql(v1.16.0)")
}

func TestNamespaceLister_Render_Table_Empty(t *testing.T) {
	t.Parallel()

	lister := &NamespaceLister{
		cfg: NamespaceListConfig{Pretty: true},
	}

	var buf bytes.Buffer
	lister.Render(&buf, []NamespaceInfo{})

	out := buf.String()
	assert.Contains(t, out, "NAMESPACE")
	assert.Contains(t, out, "MANAGED")
	assert.Contains(t, out, "OPERATORS")
	// Only header should be present, no data rows
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
}

func TestNamespaceLister_getNamespaceOperators(t *testing.T) {
	t.Parallel()

	const nsName = "db-ns"

	sub := func(name, installedCSV string) *olmv1alpha1.Subscription {
		return &olmv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: nsName},
			Status:     olmv1alpha1.SubscriptionStatus{InstalledCSV: installedCSV},
		}
	}
	csv := func(name, version string) *olmv1alpha1.ClusterServiceVersion {
		return &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: nsName},
			Spec: olmv1alpha1.ClusterServiceVersionSpec{
				Version: olmversion.OperatorVersion{Version: semver.MustParse(version)},
			},
		}
	}

	tests := []struct {
		name    string
		objects []ctrlclient.Object
		want    []string
	}{
		{
			name: "installed operator",
			objects: []ctrlclient.Object{
				sub(common.MongoDBOperatorName, "psmdb-operator.v1.20.0"),
				csv("psmdb-operator.v1.20.0", "1.20.0"),
			},
			want: []string{"psmdb(v1.20.0)"},
		},
		{
			name: "installed CSV not created yet",
			objects: []ctrlclient.Object{
				sub(common.MongoDBOperatorName, "psmdb-operator.v1.20.0"),
			},
			want: []string{},
		},
		{
			name: "installed CSV not set yet",
			objects: []ctrlclient.Object{
				sub(common.MongoDBOperatorName, ""),
			},
			want: []string{},
		},
		{
			name: "one pending, one installed",
			objects: []ctrlclient.Object{
				sub(common.MongoDBOperatorName, "psmdb-operator.v1.20.0"),
				sub(common.PostgreSQLOperatorName, "pg-operator.v2.8.0"),
				csv("pg-operator.v2.8.0", "2.8.0"),
			},
			want: []string{"postgresql(v2.8.0)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tc.objects...).
				Build()
			nsL := &NamespaceLister{
				kubeClient: kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(c),
				l:          zap.NewNop().Sugar(),
			}
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   nsName,
				Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
			}}

			got, err := nsL.getNamespaceOperators(context.Background(), ns)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
