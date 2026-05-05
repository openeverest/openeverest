package uninstall

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/percona/everest/pkg/cli/helm"
	"github.com/percona/everest/pkg/cli/steps"
	"github.com/percona/everest/pkg/common"
)

func (u *Uninstall) newStepUninstallHelmChart() steps.Step {
	return steps.Step{
		Desc: "Uninstalling Helm chart",
		F: func(ctx context.Context) error {
			return u.uninstallHelmChart(ctx)
		},
	}
}

func (u *Uninstall) newStepDeleteNamespace(ns string) steps.Step {
	return steps.Step{
		Desc: fmt.Sprintf("Deleting namespace '%s'", ns),
		F: func(ctx context.Context) error {
			return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
				delObj := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: ns},
				}
				err := u.kubeConnector.DeleteNamespace(ctx, delObj)
				if err != nil {
					if k8serrors.IsNotFound(err) {
						return true, nil
					}
					return false, err
				}
				return false, nil
			})
		},
	}
}

// helm doesn't delete CRDs by default, so we need to handle this manually.
func (u *Uninstall) newStepDeleteCRDs() steps.Step {
	return steps.Step{
		Desc: "Uninstalling CRDs",
		F: func(ctx context.Context) error {
			return u.deleteEverestCRDs(ctx)
		},
	}
}

func (u *Uninstall) deleteEverestCRDs(ctx context.Context) error {
	// We first attempt to uninstall the everest-crds chart.
	// This chart may be found only if Everest had been upgraded at least once.
	uninstaller, err := helm.NewUninstaller(
		helm.EverestCRDChartName,
		common.SystemNamespace,
		u.config.KubeconfigPath,
	)
	if err != nil {
		return fmt.Errorf("failed to create Helm uninstaller (everest-crds): %w", err)
	}
	chartFound, err := uninstaller.Uninstall(false)
	if err != nil {
		return fmt.Errorf("failed to uninstall Helm chart: %w", err)
	}
	// If this chart was not found (i.e, if no upgrade was performed), that means the CRDs
	// were installed via the everest chart. In that case, we will try to uninstall the CRDs explicitly,
	// because Helm does not uninstall CRDs by default.
	if !chartFound {
		u.l.Infof("%s chart was not installed, deleting CRDs explicitly", helm.EverestCRDChartName)
		return u.listAndDeleteEverestCRDs(ctx)
	}
	return nil
}

func (u *Uninstall) listAndDeleteEverestCRDs(ctx context.Context) error {
	crds, err := u.kubeConnector.ListCRDs(ctx, ctrlclient.MatchingLabels{common.PerconaEverestCRDLabel: "true"})
	if err != nil {
		return fmt.Errorf("failed to list CRDs: %w", err)
	}
	for _, crd := range crds.Items {
		if err := u.kubeConnector.DeleteCRD(ctx, &crd); err != nil {
			return fmt.Errorf("failed to delete CRD: %w", err)
		}
		u.l.Infof("Deleted CRD: %s", crd.GetName())
	}
	return nil
}

func (u *Uninstall) uninstallHelmChart(_ context.Context) error {
	// Delete helm chart.
	uninstaller, err := helm.NewUninstaller(common.SystemNamespace, common.SystemNamespace, u.config.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create Helm uninstaller: %w", err)
	}
	_, err = uninstaller.Uninstall(false)
	if err != nil {
		return fmt.Errorf("failed to uninstall Helm chart: %w", err)
	}
	return nil
}
