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

// Package utils provides utility functions for the Everest CLI.
package utils

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"

	goversion "github.com/hashicorp/go-version"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

const (
	dbNamespaceSubChartPath = "/charts/everest-db-namespace"
	crdSubChartPath         = "/charts/everest-crds"
)

// DBNamespaceSubChartPath returns the path to the everest-db-namespace sub-chart.
func DBNamespaceSubChartPath(dir string) string {
	return getSubChartPath(dir, dbNamespaceSubChartPath)
}

// CRDSubChartPath returns the path to the everest-crds sub-chart.
func CRDSubChartPath(dir string) string {
	return getSubChartPath(dir, crdSubChartPath)
}

func getSubChartPath(dir, subChart string) string {
	if dir == "" {
		return ""
	}
	return path.Join(dir, subChart)
}

// CheckHelmInstallation ensures that the current installation was done using Helm chart.
// Returns the version of Everest installed in the cluster.
// Returns an error if the installation was not done using Helm chart.
func CheckHelmInstallation(ctx context.Context, kubeConnector kubernetes.KubernetesConnector) (string, error) {
	everestVersion, err := version.EverestVersionFromDeployment(ctx, kubeConnector, kubeConnector.Namespace())
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", errors.New("everest is not installed in the cluster")
		}
		return "", errors.Join(err, errors.New("failed to get Everest version"))
	}

	// Versions below 1.4.0 are not installed using Helm.
	ver := everestVersion.String()
	if common.CheckConstraint(ver, "< 1.4.0") &&
		!version.IsDev(ver) { // allowed in development
		return "", errors.New("operation not supported for this version of Everest")
	}
	return ver, nil
}

// NewKubeConnector creates a new Kubernetes client.
// It auto-discovers the namespace OpenEverest is installed from the Helm release.
func NewKubeConnector(l *zap.SugaredLogger, kubeconfigPath string) (kubernetes.KubernetesConnector, error) {
	namespace, err := helm.DiscoverOpenEverestNamespace(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not discover OpenEverest namespace: %w", err)
	}
	k, err := kubernetes.New(kubeconfigPath, l, namespace)
	if err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			l.Error("Could not connect to Kubernetes. " +
				"Make sure Kubernetes is running and is accessible from this computer/server.")
		}
		return nil, err
	}
	return k, nil
}

// VerifyCLIVersion checks if the CLI version satisfies the constraints.
func VerifyCLIVersion(supVer *common.SupportedVersion) error {
	if version.Version == "" {
		return nil
	}
	cli, err := goversion.NewVersion(version.Version)
	if err != nil {
		return fmt.Errorf("failed to parse CLI version: %w", err)
	}
	if !supVer.Cli.Check(cli.Core()) {
		return fmt.Errorf(
			"cli version %q does not satisfy the constraints %q",
			cli, supVer.Cli.String(),
		)
	}
	return nil
}
