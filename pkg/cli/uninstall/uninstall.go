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

// Package uninstall ...
package uninstall

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	pollInterval = 5 * time.Second
	pollTimeout  = 5 * time.Minute
)

// Uninstall implements logic for the cluster command.
type Uninstall struct {
	config        Config
	kubeConnector kubernetes.KubernetesConnector
	l             *zap.SugaredLogger
	clusterType   kubernetes.ClusterType
}

// Config stores configuration for the Uninstall command.
type Config struct {
	// KubeconfigPath is a path to a kubeconfig
	KubeconfigPath string
	// AssumeYes is true when all questions can be skipped.
	AssumeYes bool
	// Force is true when we shall not prompt for removal.
	Force bool
	// SkipEnvDetection skips detecting the Kubernetes environment.
	SkipEnvDetection bool
	// If set, we will print the pretty output.
	Pretty bool
}

// NewUninstall returns a new Uninstall struct.
func NewUninstall(c Config, l *zap.SugaredLogger) (*Uninstall, error) {
	cli := &Uninstall{
		config: c,
		l:      l,
	}
	if c.Pretty {
		cli.l = zap.NewNop().Sugar()
	}

	kubeClient, err := cliutils.NewKubeConnector(cli.l, c.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	cli.kubeConnector = kubeClient
	return cli, nil
}

// Run runs the cluster command.
func (u *Uninstall) Run(ctx context.Context) error {
	// This command expects a Helm based installation. Otherwise, we stop here.
	// Older versions must use an older version of the CLI.
	_, err := cliutils.CheckHelmInstallation(ctx, u.kubeConnector)
	if err != nil {
		return err
	}

	if !u.config.AssumeYes {
		// user confirmation is required
		if confirm, err := u.runWizard(ctx); err != nil {
			return err
		} else if !confirm {
			u.l.Info("Exiting")
			return nil
		}
	}

	if err := u.setKubernetesEnv(ctx); err != nil {
		return fmt.Errorf("failed to detect Kubernetes environment: %w", err)
	}

	dbsExist, err := u.kubeConnector.DatabasesExist(ctx)
	if err != nil {
		return errors.Join(err, errors.New("failed to check if databases exist"))
	}

	if dbsExist && !u.config.Force {
		// there are still DB clusters managed by Everest.
		// Need to ask user for DB clusters deletion confirmation.
		if force, err := u.confirmForce(ctx); err != nil {
			return err
		} else if !force {
			u.l.Info("Can't proceed without deleting database clusters")
			return nil
		}
	}

	dbNamespaces, err := u.kubeConnector.GetDBNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database namespaces: %w", err)
	}

	uninstallSteps := u.newUninstallSteps(dbNamespaces)
	if err := steps.RunStepsWithSpinner(ctx, u.l, uninstallSteps, u.config.Pretty); err != nil {
		return err
	}

	var out io.Writer = os.Stdout
	if !u.config.Pretty {
		out = io.Discard
	}

	u.l.Infof("Everest has been uninstalled successfully")
	_, _ = fmt.Fprintln(out, "Everest has been uninstalled successfully")
	return nil
}

func (u *Uninstall) setKubernetesEnv(ctx context.Context) error {
	if !u.config.SkipEnvDetection {
		return nil
	}
	t, err := u.kubeConnector.GetClusterType(ctx)
	if err != nil {
		return err
	}
	u.clusterType = t
	u.l.Infof("Detected Kubernetes environment: %s", t)
	return nil
}

func (u *Uninstall) newUninstallSteps(nsList *corev1.NamespaceList) []steps.Step {
	var uninstallSteps []steps.Step

	if nsList != nil {
		for _, ns := range nsList.Items {
			uninstallSteps = append(uninstallSteps, namespaces.NewRemoveNamespaceSteps(ns.GetName(), false, u.kubeConnector)...)
		}
	}

	uninstallSteps = append(uninstallSteps, u.newStepUninstallHelmChart())
	uninstallSteps = append(uninstallSteps, u.newStepDeleteNamespace(common.MonitoringNamespace))
	uninstallSteps = append(uninstallSteps, u.newStepDeleteNamespace(common.SystemNamespace))
	uninstallSteps = append(uninstallSteps, u.newStepDeleteCRDs())
	return uninstallSteps
}

// Asks user for uninstall confirmation.
// Returns true if uninstall is confirmed.
func (u *Uninstall) runWizard(ctx context.Context) (bool, error) {
	msg := `You are about to uninstall Everest from the Kubernetes cluster.
This will uninstall Everest and all its components from the cluster.`
	fmt.Printf("\n%s\n\n", msg) //nolint:forbidigo

	confirm := false
	var err error
	if confirm, err = tui.NewConfirm(ctx, "Are you sure you want to uninstall Everest?").Run(); err != nil {
		return false, err
	}

	return confirm, nil
}

// Asks user for uninstall Db clusters confirmation.
// Returns true if uninstall is confirmed.
func (u *Uninstall) confirmForce(ctx context.Context) (bool, error) {
	if u.config.Force {
		return true, nil
	}

	confirm := false
	var err error
	if confirm, err = tui.NewConfirm(ctx, "There are still database clusters managed by Everest. Do you want to delete them?").Run(); err != nil {
		return false, err
	}

	return confirm, nil
}
