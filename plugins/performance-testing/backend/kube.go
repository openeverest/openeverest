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

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// newKubeClient builds a plain client-go clientset. This backend only does
// one-off Get/Create/List/Watch calls (read a credentials Secret, create a
// Job, poll its status, read its pod's logs) — no controller loop, no
// informer cache — so the typed clientset is the right tool, not
// controller-runtime's manager/cache machinery the rest of this repo's
// operators use.
//
// In-cluster config is used when running as the plugin's own Deployment
// (the real, verified deployment path). The KUBECONFIG fallback exists
// only so this binary can be run locally against a real cluster during
// development, the same way `go run ./providers/minio/cmd` was run
// directly against the target cluster for the object-storage PoC.
func newKubeClient() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			return nil, fmt.Errorf("not running in-cluster and KUBECONFIG is not set: %w", err)
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build config from KUBECONFIG: %w", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}
