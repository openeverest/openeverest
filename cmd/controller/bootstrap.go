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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/internal/controller/bootstrap"
)

const (
	bootstrapFirstRetry = time.Second
	bootstrapMostRetry  = 5 * time.Minute
	// bootstrapResync is how often the built-in manifests are applied again,
	// restoring one deleted or changed by hand.
	bootstrapResync = 10 * time.Minute

	// serviceAccountNamespaceFile is where Kubernetes mounts the namespace
	// of a pod's ServiceAccount, which is the pod's own.
	serviceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// setupBootstrap registers the runnable that creates the missing defaults
// and applies the built-in manifests. It needs the namespace the controller
// runs in, which outside a cluster only POD_NAMESPACE gives: without it,
// bootstrap is skipped.
func setupBootstrap(mgr ctrl.Manager) error {
	namespace, err := controllerNamespace(serviceAccountNamespaceFile)
	if err != nil {
		setupLog.Info("Skipping bootstrap: set POD_NAMESPACE to the namespace OpenEverest runs in", "reason", err.Error())
		return nil
	}
	// Uncached: a Get through the manager's client would start an informer
	// on every Secret and ConfigMap in the cluster.
	c, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})
	if err != nil {
		return fmt.Errorf("create the bootstrap client: %w", err)
	}
	return mgr.Add(&bootstrap.Bootstrap{
		Client:       c,
		Namespace:    namespace,
		Retry:        bootstrap.Retry{First: bootstrapFirstRetry, Most: bootstrapMostRetry},
		ResyncPeriod: bootstrapResync,
	})
}

// controllerNamespace returns POD_NAMESPACE if set, else the namespace in
// saNamespaceFile.
func controllerNamespace(saNamespaceFile string) (string, error) {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns, nil
	}
	data, err := os.ReadFile(filepath.Clean(saNamespaceFile))
	if err != nil {
		return "", err
	}
	ns := strings.TrimSpace(string(data))
	if ns == "" {
		return "", fmt.Errorf("%s is empty", saNamespaceFile)
	}
	return ns, nil
}
