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

package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// WatchBackups returns a watch.Interface that streams
// DatabaseClusterBackup events across all namespaces.
func (k *Kubernetes) WatchBackups(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &backupv1alpha1.BackupList{})
}

// WatchRestores returns a watch.Interface that streams
// DatabaseClusterRestore events across all namespaces.
func (k *Kubernetes) WatchRestores(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &backupv1alpha1.RestoreList{})
}

// WatchInstances returns a watch.Interface that streams
// Instance events across all namespaces.
func (k *Kubernetes) WatchInstances(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &corev1alpha1.InstanceList{})
}

// WatchPlugins returns a watch.Interface that streams Plugin CR events.
// Plugin is cluster-scoped.
func (k *Kubernetes) WatchPlugins(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &extensionsv1alpha1.PluginList{})
}

// WatchInstalledExtensions returns a watch.Interface that streams
// InstalledExtension events across all namespaces.
func (k *Kubernetes) WatchInstalledExtensions(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &extensionsv1alpha1.InstalledExtensionList{})
}

// WatchEverestManagedNamespaces returns a watch.Interface that streams
// events for Namespaces carrying the Everest managed-by label.
func (k *Kubernetes) WatchEverestManagedNamespaces(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &corev1.NamespaceList{},
		ctrlclient.MatchingLabels{common.KubernetesManagedByLabel: common.Everest})
}

// WatchEverestSettings returns a watch.Interface that streams ConfigMap
// events for the Everest settings ConfigMap in the system namespace.
// Callers must filter results by name (the watch returns all ConfigMaps in
// the namespace; field-selector watches are not portable across the
// controller-runtime client).
func (k *Kubernetes) WatchEverestSettings(ctx context.Context) (watch.Interface, error) {
	return k.watchList(ctx, &corev1.ConfigMapList{}, ctrlclient.InNamespace(k.Namespace()))
}

// watchList creates a controller-runtime WithWatch client and starts a watch.
func (k *Kubernetes) watchList(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) (watch.Interface, error) {
	wc, err := ctrlclient.NewWithWatch(k.restConfig, ctrlclient.Options{
		Scheme: k.k8sClient.Scheme(),
	})
	if err != nil {
		return nil, fmt.Errorf("create watch client: %w", err)
	}
	w, err := wc.Watch(ctx, list, opts...)
	if err != nil {
		return nil, fmt.Errorf("start watch: %w", err)
	}
	return w, nil
}
