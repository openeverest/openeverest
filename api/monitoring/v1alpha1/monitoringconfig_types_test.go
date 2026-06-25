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

package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMonitoringConfigList_Default(t *testing.T) {
	t.Parallel()

	now := time.Now()
	annotationKey := "openeverest.io/is-default-components-monitoring"

	testCases := []struct {
		desc     string
		list     *MonitoringConfigList
		expected *MonitoringConfig
	}{
		{
			desc:     "empty list",
			list:     &MonitoringConfigList{Items: []MonitoringConfig{}},
			expected: nil,
		},
		{
			desc: "no matching annotation",
			list: &MonitoringConfigList{
				Items: []MonitoringConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc1",
							Annotations: map[string]string{
								annotationKey: "false",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc2",
						},
					},
				},
			},
			expected: nil,
		},
		{
			desc: "single match",
			list: &MonitoringConfigList{
				Items: []MonitoringConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc1",
							Annotations: map[string]string{
								annotationKey: "true",
							},
							CreationTimestamp: metav1.NewTime(now),
						},
					},
				},
			},
			expected: &MonitoringConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mc1",
					Annotations: map[string]string{
						annotationKey: "true",
					},
					CreationTimestamp: metav1.NewTime(now),
				},
			},
		},
		{
			desc: "multiple matches, returns most recent",
			list: &MonitoringConfigList{
				Items: []MonitoringConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc1",
							Annotations: map[string]string{
								annotationKey: "true",
							},
							CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc2",
							Annotations: map[string]string{
								annotationKey: "true",
							},
							CreationTimestamp: metav1.NewTime(now),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mc3",
							Annotations: map[string]string{
								annotationKey: "true",
							},
							CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
						},
					},
				},
			},
			expected: &MonitoringConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mc2",
					Annotations: map[string]string{
						annotationKey: "true",
					},
					CreationTimestamp: metav1.NewTime(now),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			actual := tc.list.Default(annotationKey)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
