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
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

type cachedConnector struct {
	resourceVersion string
	connector       kubernetes.KubernetesConnector
}

// ClusterConnector returns the Kubernetes connector for a registered cluster.
func (h *k8sHandler) ClusterConnector(
	ctx context.Context,
	cluster string,
) (kubernetes.KubernetesConnector, error) {
	return h.connector(ctx, cluster)
}

func (h *k8sHandler) connector(
	ctx context.Context,
	cluster string,
) (kubernetes.KubernetesConnector, error) {
	// Directly constructed unit-test handlers do not have a configured cluster
	// name; production always has one from EverestConfig.
	if h.clusterName == "" || cluster == h.clusterName {
		return h.kubeConnector, nil
	}

	connection, err := h.kubeConnector.GetClusterConnection(
		ctx,
		types.NamespacedName{Name: cluster},
	)
	if err != nil {
		return nil, fmt.Errorf("cluster %q is not registered: %w", cluster, err)
	}
	credentialsRef := connection.Spec.CredentialsSecretRef
	credentials, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{
		Namespace: credentialsRef.Namespace,
		Name:      credentialsRef.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster %q credentials: %w", cluster, err)
	}
	token := credentials.Data["token"]
	if len(token) == 0 {
		return nil, fmt.Errorf("cluster %q credential Secret has no token", cluster)
	}

	resourceVersion := connection.ResourceVersion + ":" + credentials.ResourceVersion
	h.remoteMu.Lock()
	defer h.remoteMu.Unlock()
	if cached, ok := h.remoteConnectors[cluster]; ok &&
		cached.resourceVersion == resourceVersion {
		return cached.connector, nil
	}

	remote, err := kubernetes.NewRemote(kubernetes.RemoteClusterConfig{
		Endpoint:    connection.Spec.Endpoint,
		BearerToken: string(token),
		CAData:      credentials.Data["ca.crt"],
	}, h.log, h.kubeConnector.Namespace())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster %q: %w", cluster, err)
	}
	h.remoteConnectors[cluster] = cachedConnector{
		resourceVersion: resourceVersion,
		connector:       remote,
	}
	return remote, nil
}

func (h *k8sHandler) invalidateConnector(cluster string) {
	h.remoteMu.Lock()
	delete(h.remoteConnectors, cluster)
	h.remoteMu.Unlock()
}
