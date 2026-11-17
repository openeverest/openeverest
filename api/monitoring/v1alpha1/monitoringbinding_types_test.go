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

func TestConfiguredReasonStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason monitoringv1alpha1.ConfiguredReason
		want   metav1.ConditionStatus
	}{
		{monitoringv1alpha1.ReasonConfigured, metav1.ConditionTrue},
		{monitoringv1alpha1.ReasonMaintenancePending, metav1.ConditionTrue},
		{monitoringv1alpha1.ReasonPending, metav1.ConditionUnknown},
		{monitoringv1alpha1.ReasonRetained, metav1.ConditionUnknown},
		{monitoringv1alpha1.ReasonNotAccepted, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonRenderFailed, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonApplyFailed, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonAgentNotReady, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonEndpointNotServing, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonServerTooOld, metav1.ConditionFalse},
		{monitoringv1alpha1.ReasonNotSelected, metav1.ConditionFalse},
		{monitoringv1alpha1.ConfiguredReason("SomethingNew"), metav1.ConditionFalse},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.reason.Status())
		})
	}
}
