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

package rbac

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func newConfigMapPolicy(policy string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      common.EverestRBACConfigMapName,
		},
		Data: map[string]string{
			"enabled":    "true",
			"policy.csv": policy,
		},
	}
}

func newConfigMapMock(policy string) kubernetes.KubernetesConnector {
	mockClient := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(newConfigMapPolicy(policy))
	return kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-ns").WithKubernetesClient(mockClient.Build())
}

func newPolicy(lines ...string) string {
	return strings.Join(lines, "\n")
}

func testUserGetter(ctx context.Context) (rbac.User, error) {
	user, ok := ctx.Value(common.UserCtxKey).(rbac.User)
	if !ok {
		return rbac.User{}, errors.New("user not found in context")
	}
	return user, nil
}
