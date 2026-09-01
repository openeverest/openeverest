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
	"testing"

	"github.com/stretchr/testify/assert"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/percona/everest/pkg/common"
)

func TestIsManagedByEverest(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name     string
		input    *v1.Namespace
		expected bool
	}

	tcases := []tcase{
		{
			name: "managed by everest",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
					Labels: map[string]string{
						common.KubernetesManagedByLabel: common.Everest,
					},
				},
			},
			expected: true,
		},
		{
			name: "managed by other",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
					Labels: map[string]string{
						common.KubernetesManagedByLabel: "helm",
					},
				},
			},
			expected: false,
		},
		{
			name: "label not present",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-ns",
					Labels: make(map[string]string),
				},
			},
			expected: false,
		},
		{
			name: "nil labels",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			expected: false,
		},
		{
			name: "empty label value",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
					Labels: map[string]string{
						common.KubernetesManagedByLabel: "",
					},
				},
			},
			expected: false,
		},
		{
			name: "extra labels present",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
					Labels: map[string]string{
						common.KubernetesManagedByLabel: common.Everest,
						"other-label":                   "other-value",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isManagedByEverest(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEnsureNoOperatorsRemoved(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name          string
		subscriptions []olmv1alpha1.Subscription
		installPG     bool
		installPXC    bool
		installPSMDB  bool
		expected      bool
	}

	tcases := []tcase{
		{
			name:          "no subscriptions, all flags false - allowed",
			subscriptions: []olmv1alpha1.Subscription{},
			installPG:     false,
			installPXC:    false,
			installPSMDB:  false,
			expected:      true,
		},
		{
			name:          "no subscriptions, all flags true - allowed",
			subscriptions: []olmv1alpha1.Subscription{},
			installPG:     true,
			installPXC:    true,
			installPSMDB:  true,
			expected:      true,
		},
		{
			name: "pg installed, pg flag true - allowed",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
			},
			installPG: true,
			expected:  true,
		},
		{
			name: "pg installed, pg flag false - blocked",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
			},
			installPG: false,
			expected:  false,
		},
		{
			name: "pxc installed, pxc flag true - allowed",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.MySQLOperatorName}},
			},
			installPXC: true,
			expected:   true,
		},
		{
			name: "pxc installed, pxc flag false - blocked",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.MySQLOperatorName}},
			},
			installPXC: false,
			expected:   false,
		},
		{
			name: "psmdb installed, psmdb flag true - allowed",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.MongoDBOperatorName}},
			},
			installPSMDB: true,
			expected:     true,
		},
		{
			name: "psmdb installed, psmdb flag false - blocked",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.MongoDBOperatorName}},
			},
			installPSMDB: false,
			expected:     false,
		},
		{
			name: "all operators installed, all flags true - allowed",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.MySQLOperatorName}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.MongoDBOperatorName}},
			},
			installPG:    true,
			installPXC:   true,
			installPSMDB: true,
			expected:     true,
		},
		{
			name: "all operators installed, one flag false - blocked",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.MySQLOperatorName}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.MongoDBOperatorName}},
			},
			installPG:    true,
			installPXC:   false,
			installPSMDB: true,
			expected:     false,
		},
		{
			name: "unknown operator subscription is ignored",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: "unknown-operator"}},
			},
			installPG:    false,
			installPXC:   false,
			installPSMDB: false,
			expected:     true,
		},
		{
			name: "mixed known and unknown subscriptions",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: "custom-operator"}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
			},
			installPG: true,
			expected:  true,
		},
		{
			name: "mixed known and unknown, known removed - blocked",
			subscriptions: []olmv1alpha1.Subscription{
				{ObjectMeta: metav1.ObjectMeta{Name: "custom-operator"}},
				{ObjectMeta: metav1.ObjectMeta{Name: common.PostgreSQLOperatorName}},
			},
			installPG: false,
			expected:  false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ensureNoOperatorsRemoved(
				tc.subscriptions,
				tc.installPG,
				tc.installPXC,
				tc.installPSMDB,
			)
			assert.Equal(t, tc.expected, result)
		})
	}
}
