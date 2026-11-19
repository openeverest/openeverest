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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestClusterIdentityComesFromConfiguration(t *testing.T) {
	t.Parallel()

	client := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).Build()
	handler := &k8sHandler{
		clusterName:   "prod-eu",
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(client),
	}

	list, err := handler.ListClusters(t.Context())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, "prod-eu", list.Items[0].Name)

	cluster, err := handler.GetCluster(t.Context(), "prod-eu")
	require.NoError(t, err)
	require.Equal(t, "prod-eu", cluster.Name)

	_, err = handler.GetCluster(t.Context(), "other")
	require.ErrorContains(t, err, `cluster "other" not found`)
}

func TestRegisterAndDeleteCluster(t *testing.T) {
	t.Parallel()

	client := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).Build()
	handler := &k8sHandler{
		clusterName:      "local",
		kubeConnector:    kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(client),
		remoteConnectors: make(map[string]cachedConnector),
		log:              zap.NewNop().Sugar(),
	}
	token := "secret-token"

	registered, err := handler.RegisterCluster(t.Context(), &api.ClusterRegistration{
		Name:        "prod-eu",
		Server:      "https://agent.example.test",
		BearerToken: &token,
	})
	require.NoError(t, err)
	require.Equal(t, "prod-eu", registered.Name)

	connection, err := handler.kubeConnector.GetClusterConnection(
		t.Context(),
		ctrlclient.ObjectKey{Name: "prod-eu"},
	)
	require.NoError(t, err)
	secret, err := handler.kubeConnector.GetSecret(
		t.Context(),
		ctrlclient.ObjectKey{
			Namespace: connection.Spec.CredentialsSecretRef.Namespace,
			Name:      connection.Spec.CredentialsSecretRef.Name,
		},
	)
	require.NoError(t, err)
	require.Equal(t, token, string(secret.Data["token"]))

	require.NoError(t, handler.DeleteCluster(t.Context(), "prod-eu"))
	_, err = handler.kubeConnector.GetClusterConnection(
		t.Context(),
		ctrlclient.ObjectKey{Name: "prod-eu"},
	)
	require.Error(t, err)
	_, err = handler.kubeConnector.GetSecret(
		t.Context(),
		ctrlclient.ObjectKey{
			Namespace: connection.Spec.CredentialsSecretRef.Namespace,
			Name:      connection.Spec.CredentialsSecretRef.Name,
		},
	)
	require.Error(t, err)
}

func TestConnectorResolvesRemoteRegistration(t *testing.T) {
	t.Parallel()

	objects := []ctrlclient.Object{
		&extensionsv1alpha1.ClusterConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "prod-eu",
				ResourceVersion: "10",
			},
			Spec: extensionsv1alpha1.ClusterConnectionSpec{
				Endpoint: "https://agent.example.test",
				CredentialsSecretRef: common.NamespacedObjectRef{
					Namespace: "everest-system",
					Name:      "agent-credentials",
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "everest-system",
				Name:            "agent-credentials",
				ResourceVersion: "11",
			},
			Data: map[string][]byte{"token": []byte("hub-token")},
		},
	}
	client := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(objects...).
		Build()
	handler := &k8sHandler{
		clusterName:      "local",
		kubeConnector:    kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(client),
		remoteConnectors: make(map[string]cachedConnector),
		log:              zap.NewNop().Sugar(),
	}

	first, err := handler.ClusterConnector(context.Background(), "prod-eu")
	require.NoError(t, err)
	second, err := handler.ClusterConnector(context.Background(), "prod-eu")
	require.NoError(t, err)
	require.Same(t, first, second)
	require.Equal(t, "https://agent.example.test", first.Config().Host)
	require.Equal(t, "hub-token", first.Config().BearerToken)
}
