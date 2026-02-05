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

// Package namespaces provides the functionality to manage namespaces.
package namespaces

import (
	"context"
	"fmt"
	"slices"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	everestOperator "github.com/percona/everest-operator/api/everest/v1alpha1"
	cliutils "github.com/percona/everest/pkg/cli/utils"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

// skipNamespaces is a list of namespaces that cannot be added to Everest management.
// It contains Kubernetes system, reserved by Everest core and cloud providers specific namespaces.
// Note: this list is not exhaustive and can be extended.
var skipNamespaces = []string{
	// Kubernetes native system namespaces.
	"kube-system",
	"kube-public",
	"kube-node-lease",

	// Everest core namespaces.
	common.SystemNamespace,
	common.MonitoringNamespace,
	kubernetes.OLMNamespace,

	// GKE namespaces.
	"gke-managed-cim",
	"gke-managed-system",
	"gke-managed-volumepopulator",
	"gmp-public",
	"gmp-system",
}

type (
	// NamespaceListConfig is the configuration for the  namespace listing operation.
	NamespaceListConfig struct {
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string
		// Pretty if set print the output in pretty mode.
		Pretty bool
		// ListAllNamespaces if set, list all namespaces.
		// Note: this flag skips namespaces that cannot be added to Everest management
		// (i.e. Kubernetes system, specific to cloud providers and Everest Core namespaces).
		ListAllNamespaces bool
	}

	// NamespaceInfo contains information about a namespace.
	NamespaceInfo struct {
		// Name is the namespace name.
		Name string
		// InstalledOperators is a list of installed Percona operators in the namespace.
		InstalledOperators []string
	}

	// NamespaceLister is the CLI operation to list namespaces.
	NamespaceLister struct {
		cfg        NamespaceListConfig
		kubeClient kubernetes.KubernetesConnector
		l          *zap.SugaredLogger
	}
)

// NewNamespaceLister returns a new CLI operation to list namespaces.
func NewNamespaceLister(c NamespaceListConfig, l *zap.SugaredLogger) (*NamespaceLister, error) {
	n := &NamespaceLister{
		cfg: c,
		l:   l.With("component", "namespace-lister"),
	}
	if c.Pretty {
		n.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(n.l, n.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	n.kubeClient = k
	return n, nil
}

// Run the namespace list operation.
func (nsL *NamespaceLister) Run(ctx context.Context) ([]NamespaceInfo, error) {
	var err error
	// This command expects a Helm based installation (< 1.4.0)
	_, err = cliutils.CheckHelmInstallation(ctx, nsL.kubeClient)
	if err != nil {
		return nil, err
	}

	var nsList *corev1.NamespaceList
	opts := []client.ListOption{client.MatchingFields{"status.phase": string(corev1.NamespaceActive)}}

	if !nsL.cfg.ListAllNamespaces {
		// show only namespaces already managed by Everest.
		opts = append(opts, client.MatchingLabels{common.KubernetesManagedByLabel: common.Everest})
	}

	if nsList, err = nsL.kubeClient.ListNamespaces(ctx, opts...); err != nil {
		return nil, err
	}

	// filter out namespaces that are listed in skipNamespaces and non-active namespaces.
	nsList.Items = slices.DeleteFunc(nsList.Items, func(ns corev1.Namespace) bool {
		return slices.Contains(skipNamespaces, ns.Name)
	})

	var toReturn []NamespaceInfo
	for _, ns := range nsList.Items {
		nsInfo := NamespaceInfo{Name: ns.GetName()}
		if nsInfo.InstalledOperators, err = nsL.getNamespaceOperators(ctx, &ns); err != nil {
			return nil, fmt.Errorf("cannot get namespace subscriptions: %w", err)
		}
		slices.Sort(nsInfo.InstalledOperators)
		toReturn = append(toReturn, nsInfo)
	}
	return toReturn, nil
}

// getNamespaceOperators returns a list of installed operators in the namespace.
// It returns an empty list if the namespace is not managed by Everest.
func (nsL *NamespaceLister) getNamespaceOperators(ctx context.Context, ns *v1.Namespace) ([]string, error) {
	var toReturn []string
	if isManagedByEverest(ns) {
		// no need to look for installed operators from namespaces not managed by Everest.
		subList, err := nsL.kubeClient.ListInstalledOperators(ctx, client.InNamespace(ns.GetName()))
		if err != nil && client.IgnoreNotFound(err) != nil {
			return []string{}, fmt.Errorf("cannot list installed operators in namespace='%s': %w", ns.GetName(), err)
		}

		for _, sub := range subList.Items {
			csv, csvErr := nsL.kubeClient.GetClusterServiceVersion(ctx, types.NamespacedName{
				Namespace: ns.GetName(),
				Name:      sub.Status.InstalledCSV,
			})
			if csvErr != nil && client.IgnoreNotFound(csvErr) != nil {
				return []string{}, fmt.Errorf("cannot list installed operators in namespace='%s': %w", ns.GetName(), csvErr)
			}
			v, err := goversion.NewVersion(csv.Spec.Version.FinalizeVersion())
			if err != nil {
				return []string{}, fmt.Errorf("cannot parse operator='%s' version in namespace='%s': %w",
					sub.Spec.CatalogSourceNamespace,
					ns.GetName(),
					err,
				)
			}
			toReturn = append(toReturn, fmt.Sprintf("%s(v%s)", convertDbOperatorName(sub.GetName()), v.String()))
		}
	}
	return toReturn, nil
}

func convertDbOperatorName(name string) string {
	switch strings.ToLower(name) {
	case common.MongoDBOperatorName:
		return string(everestOperator.DatabaseEnginePSMDB)
	case common.MySQLOperatorName:
		return string(everestOperator.DatabaseEnginePXC)
	case common.PostgreSQLOperatorName:
		return string(everestOperator.DatabaseEnginePostgresql)
	default:
		return name
	}
}
