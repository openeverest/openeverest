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

// Package namespaces provides the functionality to manage namespaces.
package namespaces

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	pollInterval = 5 * time.Second
	pollTimeout  = 5 * time.Minute
)

// NamespaceRemoveConfig is the configuration for the namespace removal operation.
type (
	NamespaceRemoveConfig struct {
		// NamespaceList is a list of namespaces to be removed.
		// The property shall be set explicitly after the provided namespaces are parsed and validated using ValidateNamespaces func.
		NamespaceList []string
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string
		// Force delete a namespace by deleting databases in it.
		Force bool
		// If set, keep the namespace but remove all resources from it.
		KeepNamespace bool
		// Pretty if set print the output in pretty mode.
		Pretty bool
	}

	// NamespaceRemover is the CLI operation to remove namespaces.
	NamespaceRemover struct {
		cfg           NamespaceRemoveConfig
		kubeConnector kubernetes.KubernetesConnector
		l             *zap.SugaredLogger
	}
)

// ValidateNamespaces validates the provided list of namespaces.
// It validates:
// - namespace names
// - namespace ownership
func (cfg *NamespaceRemoveConfig) ValidateNamespaces(ctx context.Context, nsList []string) error {
	if err := validateNamespaceNames(nsList); err != nil {
		return err
	}

	k, err := cliutils.NewKubeConnector(zap.NewNop().Sugar(), cfg.KubeconfigPath)
	if err != nil {
		return err
	}

	for _, ns := range nsList {
		if err := cfg.validateNamespaceOwnership(ctx, k, ns); err != nil {
			return err
		}
		// if --force flag is passed - it doesn't matter if there are DB clusters in the namespace.
		if !cfg.Force {
			// Check that there are no DB clusters left in namespaces.
			if dbsExist, err := k.DatabasesExist(ctx, ctrlclient.InNamespace(ns)); err != nil {
				return errors.Join(err, fmt.Errorf("failed to check if databases exist in namespace='%s'", ns))
			} else if dbsExist {
				return ErrNamespaceNotEmpty
			}
		}
	}

	return nil
}

// validateNamespaceOwnership validates the namespace existence and ownership.
func (cfg *NamespaceRemoveConfig) validateNamespaceOwnership(
	ctx context.Context,
	k kubernetes.KubernetesConnector,
	namespace string,
) error {
	// Check that the namespace exists.
	exists, managedByEverest, err := namespaceExists(ctx, k, namespace)
	if err != nil {
		return err
	}
	if !exists {
		return NewErrNamespaceNotExist(namespace)
	}

	if !managedByEverest {
		return NewErrNamespaceNotManagedByEverest(namespace)
	}

	return nil
}

// NewNamespaceRemove returns a new CLI operation to remove namespaces.
func NewNamespaceRemove(c NamespaceRemoveConfig, l *zap.SugaredLogger) (*NamespaceRemover, error) {
	n := &NamespaceRemover{
		cfg: c,
		l:   l.With("component", "namespace-remover"),
	}
	if c.Pretty {
		n.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(n.l, n.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	n.kubeConnector = k
	return n, nil
}

// Run the namespace removal operation.
func (r *NamespaceRemover) Run(ctx context.Context) error {
	// This command expects a Helm based installation (< 1.4.0)
	_, err := cliutils.CheckHelmInstallation(ctx, r.kubeConnector)
	if err != nil {
		return err
	}

	var removalSteps []steps.Step
	for _, ns := range r.cfg.NamespaceList {
		removalSteps = append(removalSteps, NewRemoveNamespaceSteps(ns, r.cfg.KeepNamespace, r.kubeConnector)...)
	}

	return steps.RunStepsWithSpinner(ctx, r.l, removalSteps, r.cfg.Pretty)
}

// NewRemoveNamespaceSteps returns the steps to remove a namespace.
func NewRemoveNamespaceSteps(namespace string, keepNs bool, k kubernetes.KubernetesConnector) []steps.Step {
	removeSteps := []steps.Step{
		{
			Desc: fmt.Sprintf("Deleting database clusters in namespace '%s'", namespace),
			F: func(ctx context.Context) error {
				return k.DeleteDatabaseClusters(ctx, ctrlclient.InNamespace(namespace))
			},
		},
		{
			Desc: fmt.Sprintf("Deleting backup storages in namespace '%s'", namespace),
			F: func(ctx context.Context) error {
				return k.DeleteBackupStorages(ctx, ctrlclient.InNamespace(namespace))
			},
		},
		{
			Desc: fmt.Sprintf("Deleting monitoring instances in namespace '%s'", namespace),
			F: func(ctx context.Context) error {
				return k.DeleteMonitoringConfigs(ctx, ctrlclient.InNamespace(namespace))
			},
		},
	}
	nsStepDesc := fmt.Sprintf("Deleting database namespace '%s'", namespace)
	if keepNs {
		nsStepDesc = fmt.Sprintf("Deleting resources from namespace '%s'", namespace)
	}
	removeSteps = append(removeSteps, steps.Step{
		Desc: nsStepDesc,
		F: func(ctx context.Context) error {
			u, err := helm.NewUninstaller(namespace, namespace, k.Kubeconfig())
			if err != nil {
				return errors.Join(err, errors.New("failed to create helm uninstaller"))
			}
			if _, err := u.Uninstall(false); err != nil {
				return errors.Join(err, errors.New("failed to uninstall helm chart"))
			}
			if keepNs {
				// keep the namespace, but remove the Everest label
				return removeEverestLabelFromNamespace(ctx, k, namespace)
			}
			delObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}
			if err := k.DeleteNamespace(ctx, delObj); err != nil {
				return err
			}
			return ensureNamespaceGone(ctx, namespace, k)
		},
	})
	return removeSteps
}

func removeEverestLabelFromNamespace(ctx context.Context, k kubernetes.KubernetesConnector, namespace string) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, false, func(ctx context.Context) (bool, error) {
		ns, err := k.GetNamespace(ctx, types.NamespacedName{Name: namespace})
		if err != nil {
			return true, err
		}
		if !isManagedByEverest(ns) {
			return true, nil
		}
		labels := ns.GetLabels()
		delete(labels, common.KubernetesManagedByLabel)
		ns.SetLabels(labels)
		_, err = k.UpdateNamespace(ctx, ns)
		if err != nil && k8serrors.IsConflict(err) {
			return false, nil
		}
		return true, err
	})
}

func ensureNamespaceGone(ctx context.Context, namespace string, k kubernetes.KubernetesConnector) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, false, func(ctx context.Context) (bool, error) {
		_, err := k.GetNamespace(ctx, types.NamespacedName{Name: namespace})
		if err != nil && k8serrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	})
}
