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

package helm

import (
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
)

// DiscoverOpenEverestNamespace returns the namespace where OpenEverest is installed.
// It scans all deployed Helm releases for the OpenEverest chart because the
// namespace is user-chosen at install time and not stored in a fixed location.
func DiscoverOpenEverestNamespace(kubeconfigPath string) (string, error) {
	cfg, err := newActionsCfg("", kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to create Helm configuration: %w", err)
	}

	return discoverOpenEverestNamespace(cfg)
}

// discoverOpenEverestNamespace lists all deployed Helm releases and returns the
// namespace of the one whose chart name matches EverestChartName.
func discoverOpenEverestNamespace(cfg *action.Configuration) (string, error) {
	list := action.NewList(cfg)
	list.AllNamespaces = true
	list.StateMask = action.ListDeployed

	releases, err := list.Run()
	if err != nil {
		return "", fmt.Errorf("failed to list Helm releases: %w", err)
	}

	for _, rel := range releases {
		if rel.Chart != nil && rel.Chart.Metadata != nil && rel.Chart.Metadata.Name == EverestChartName {
			return rel.Namespace, nil
		}
	}

	return "", errors.New("no OpenEverest Helm release found; is OpenEverest installed?")
}
