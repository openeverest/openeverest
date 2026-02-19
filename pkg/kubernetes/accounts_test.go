// everest
// Copyright (C) 2025 Percona LLC
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
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestAccounts(t *testing.T) {
	t.Parallel()

	objs := []ctrlclient.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: common.SystemNamespace},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.EverestAccountsSecretName,
				Namespace: common.SystemNamespace,
			},
		},
	}

	mockClient := fakeclient.NewClientBuilder().WithScheme(CreateScheme())
	mockClient.WithObjects(objs...)
	k := NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
	accounts.Tests(t, k.Accounts())
}
