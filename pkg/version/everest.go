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

package version

import (
	"context"
	"errors"
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

type deploymentGetter interface {
	GetDeployment(ctx context.Context, key ctrlclient.ObjectKey) (*appsv1.Deployment, error)
}

// EverestVersionFromDeployment returns Everest version from the k8s deployment resource.
func EverestVersionFromDeployment(ctx context.Context, dg deploymentGetter) (*goversion.Version, error) {
	dep, err := dg.GetDeployment(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.PerconaEverestDeploymentName})
	// Ignore not found error, we will try again with legacy name.
	if ctrlclient.IgnoreNotFound(err) != nil {
		return nil, err
	}

	// If the deployment is not found, try to get it with the legacy name.
	if dep == nil || dep.GetCreationTimestamp().Time.IsZero() {
		dep, err = dg.GetDeployment(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.PerconaEverestDeploymentNameLegacy})
		if err != nil {
			return nil, err
		}
	}

	var container *corev1.Container
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Name == common.EverestContainerNameInDeployment {
			//nolint:exportloopref
			container = &c
			break
		}
	}

	if container == nil {
		return nil, errors.New("everest container not found in deployment")
	}

	image := container.Image
	versionSplit := ":"
	i := strings.LastIndex(image, versionSplit)
	if i == -1 {
		return nil, fmt.Errorf("everest version cannot be determined from %q", image)
	}

	cutIndex := i + len(versionSplit)
	versionTag, _ := strings.CutPrefix(image[cutIndex:], "v")

	v, err := goversion.NewVersion(versionTag)
	if err != nil {
		return nil, err
	}

	return v, nil
}
