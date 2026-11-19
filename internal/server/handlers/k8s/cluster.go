// everest
// Copyright (C) 2023 Percona LLC
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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

const localClusterServer = "https://kubernetes.default.svc"

// ListClusters returns a list of clusters.
func (h *k8sHandler) ListClusters(ctx context.Context) (*api.ClusterList, error) {
	remotes, err := h.kubeConnector.ListClusterConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster connections: %w", err)
	}

	clusters := &api.ClusterList{
		Items: make([]api.Cluster, 0, len(remotes.Items)+1),
	}
	clusters.Items = append(clusters.Items, api.Cluster{
		Name:   h.clusterName,
		Server: localClusterServer,
	})
	for _, connection := range remotes.Items {
		clusters.Items = append(clusters.Items, api.Cluster{
			Name:   connection.Name,
			Server: connection.Spec.Endpoint,
		})
	}
	return clusters, nil
}

// GetCluster returns a cluster by name.
func (h *k8sHandler) GetCluster(ctx context.Context, name string) (*api.Cluster, error) {
	if name == h.clusterName {
		return &api.Cluster{Name: h.clusterName, Server: localClusterServer}, nil
	}

	connection, err := h.kubeConnector.GetClusterConnection(ctx, types.NamespacedName{Name: name})
	if err != nil {
		return nil, fmt.Errorf("cluster %q not found: %w", name, err)
	}
	return &api.Cluster{
		Name:   connection.Name,
		Server: connection.Spec.Endpoint,
	}, nil
}

// RegisterCluster persists a remote agent connection and its credentials.
func (h *k8sHandler) RegisterCluster(
	ctx context.Context,
	registration *api.ClusterRegistration,
) (*api.Cluster, error) {
	if registration == nil {
		return nil, errors.New("cluster registration is required")
	}
	if registration.Name == h.clusterName {
		return nil, fmt.Errorf("cluster %q is the local cluster", registration.Name)
	}
	if errs := k8svalidation.IsDNS1123Subdomain(registration.Name); len(errs) != 0 {
		return nil, fmt.Errorf("invalid cluster name %q: %s", registration.Name, errs[0])
	}
	endpoint, err := url.Parse(registration.Server)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" {
		return nil, fmt.Errorf("cluster %q server must be an HTTPS URL", registration.Name)
	}
	if registration.BearerToken == nil || *registration.BearerToken == "" {
		return nil, errors.New("bearerToken is required")
	}

	secretName := clusterCredentialSecretName(registration.Name)
	secretData := map[string][]byte{"token": []byte(*registration.BearerToken)}
	if registration.CaData != nil {
		secretData["ca.crt"] = *registration.CaData
	}
	secret, err := h.kubeConnector.CreateSecret(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: h.kubeConnector.Namespace(),
			Labels: map[string]string{
				"openeverest.io/cluster-connection": registration.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store cluster credentials: %w", err)
	}

	_, err = h.kubeConnector.CreateClusterConnection(ctx, &extensionsv1alpha1.ClusterConnection{
		ObjectMeta: metav1.ObjectMeta{Name: registration.Name},
		Spec: extensionsv1alpha1.ClusterConnectionSpec{
			Endpoint: registration.Server,
			CredentialsSecretRef: common.NamespacedObjectRef{
				Namespace: secret.Namespace,
				Name:      secret.Name,
			},
		},
	})
	if err != nil {
		rollbackErr := h.kubeConnector.DeleteSecret(ctx, secret)
		return nil, errors.Join(
			fmt.Errorf("failed to register cluster: %w", err),
			rollbackErr,
		)
	}
	h.invalidateConnector(registration.Name)
	return &api.Cluster{Name: registration.Name, Server: registration.Server}, nil
}

// DeleteCluster removes a remote registration and its credentials.
func (h *k8sHandler) DeleteCluster(ctx context.Context, name string) error {
	if name == h.clusterName {
		return fmt.Errorf("cluster %q is the local cluster and cannot be unregistered", name)
	}
	connection, err := h.kubeConnector.GetClusterConnection(ctx, types.NamespacedName{Name: name})
	if err != nil {
		return fmt.Errorf("cluster %q not found: %w", name, err)
	}
	if err := h.kubeConnector.DeleteClusterConnection(ctx, connection); err != nil {
		return fmt.Errorf("failed to unregister cluster %q: %w", name, err)
	}
	h.invalidateConnector(name)

	credentialsRef := connection.Spec.CredentialsSecretRef
	if err := h.kubeConnector.DeleteSecret(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: credentialsRef.Namespace,
			Name:      credentialsRef.Name,
		},
	}); err != nil {
		return fmt.Errorf("cluster was unregistered but credentials could not be deleted: %w", err)
	}
	return nil
}

func clusterCredentialSecretName(cluster string) string {
	sum := sha256.Sum256([]byte(cluster))
	return "cluster-connection-" + hex.EncodeToString(sum[:8])
}
