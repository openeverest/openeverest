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

package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func TestReadyReasonStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason monitoringv1alpha1.ReadyReason
		want   metav1.ConditionStatus
	}{
		{monitoringv1alpha1.ReasonReady, metav1.ConditionTrue},
		{monitoringv1alpha1.ReasonReadyPending, metav1.ConditionUnknown},
		{monitoringv1alpha1.ReasonUnreachable, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonUnauthorized, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonUnsupportedServerVersion, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonReadyInvalidParameters, metav1.ConditionFalse},
		{monitoringv1alpha1.ReadyReason("SomethingNew"), metav1.ConditionFalse},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.reason.Status())
		})
	}
}
