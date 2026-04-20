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

// Package status provides the logic for the `status` command.
package status

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/output"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

// StatusConfig holds the configuration for the `status` command.
type StatusConfig struct {
	// KubeconfigPath is the path to the kubeconfig file.
	KubeconfigPath string
	// Pretty if set print the output in pretty mode.
	Pretty bool
	// JSON if set print the output in JSON mode.
	JSON bool
}

// Status provides the functionality to check Everest health.
type Status struct {
	l          *zap.SugaredLogger
	cfg        StatusConfig
	kubeClient kubernetes.KubernetesConnector
}

// ComponentStatus represents the health status of a single component.
type ComponentStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Ready     bool   `json:"ready"`
	Message   string `json:"message,omitempty"`
}

// OperatorStatus represents the health status of a database operator.
type OperatorStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version,omitempty"`
	Ready     bool   `json:"ready"`
	Message   string `json:"message,omitempty"`
}

// OverallStatus represents the overall health status of Everest.
type OverallStatus struct {
	EverestVersion string            `json:"everestVersion"`
	Healthy        bool              `json:"healthy"`
	Components     []ComponentStatus `json:"components"`
	Operators      []OperatorStatus  `json:"operators"`
	Namespaces     []string          `json:"namespaces"`
}

// NewStatus returns a new Status struct.
func NewStatus(cfg StatusConfig, l *zap.SugaredLogger) (*Status, error) {
	s := &Status{
		l:   l.With("component", "status"),
		cfg: cfg,
	}
	if cfg.Pretty {
		s.l = zap.NewNop().Sugar()
	}

	var err error
	s.kubeClient, err = cliutils.NewKubeConnector(s.l, cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Run executes the status check.
func (s *Status) Run(ctx context.Context) error {
	result := &OverallStatus{
		Healthy: true,
	}

	// Get Everest version.
	ev, err := version.EverestVersionFromDeployment(ctx, s.kubeClient)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			result.EverestVersion = "[NOT INSTALLED]"
			result.Healthy = false
		} else {
			return fmt.Errorf("failed to get Everest version: %w", err)
		}
	} else {
		result.EverestVersion = fmt.Sprintf("v%s", ev.String())
	}

	// Check core components.
	s.checkCoreComponents(ctx, result)

	// Check OLM components.
	s.checkOLMComponents(ctx, result)

	// Check monitoring components.
	s.checkMonitoringComponents(ctx, result)

	// Check database operators.
	s.checkDatabaseOperators(ctx, result)

	// Get managed namespaces.
	s.getManagedNamespaces(ctx, result)

	// Print results.
	if s.cfg.JSON {
		return s.printJSON(result)
	}
	s.printPretty(result)
	return nil
}

func (s *Status) checkCoreComponents(ctx context.Context, result *OverallStatus) {
	coreDeployments := []struct {
		name      string
		namespace string
	}{
		{common.PerconaEverestDeploymentName, common.SystemNamespace},
		{common.PerconaEverestOperatorDeploymentName, common.SystemNamespace},
	}

	for _, d := range coreDeployments {
		cs := s.checkDeployment(ctx, d.name, d.namespace)
		result.Components = append(result.Components, cs)
		if !cs.Ready {
			result.Healthy = false
		}
	}
}

func (s *Status) checkOLMComponents(ctx context.Context, result *OverallStatus) {
	depls, err := s.kubeClient.ListDeployments(ctx, ctrlclient.InNamespace(kubernetes.OLMNamespace))
	if err != nil {
		s.l.Debugf("Failed to list OLM deployments: %v", err)
		result.Components = append(result.Components, ComponentStatus{
			Name:      "OLM",
			Namespace: kubernetes.OLMNamespace,
			Ready:     false,
			Message:   fmt.Sprintf("failed to list deployments: %v", err),
		})
		result.Healthy = false
		return
	}

	for _, depl := range depls.Items {
		cs := s.checkDeployment(ctx, depl.GetName(), depl.GetNamespace())
		result.Components = append(result.Components, cs)
		if !cs.Ready {
			result.Healthy = false
		}
	}
}

func (s *Status) checkMonitoringComponents(ctx context.Context, result *OverallStatus) {
	monitoringDeployments := []string{
		common.VictoriaMetricsOperatorDeploymentName,
		common.KubeStateMetricsDeploymentName,
	}

	for _, name := range monitoringDeployments {
		cs := s.checkDeployment(ctx, name, common.MonitoringNamespace)
		result.Components = append(result.Components, cs)
		if !cs.Ready {
			result.Healthy = false
		}
	}
}

func (s *Status) checkDatabaseOperators(ctx context.Context, result *OverallStatus) {
	operators := []struct {
		product string
		name    string
	}{
		{common.MySQLProductName, common.MySQLOperatorName},
		{common.MongoDBProductName, common.MongoDBOperatorName},
		{common.PostgreSQLProductName, common.PostgreSQLOperatorName},
	}

	// Get all DB namespaces to check for operators.
	dbNamespaces, err := s.kubeClient.GetDBNamespaces(ctx)
	if err != nil {
		s.l.Debugf("Failed to get DB namespaces: %v", err)
		return
	}

	for _, ns := range dbNamespaces.Items {
		for _, op := range operators {
			os := OperatorStatus{
				Name:      fmt.Sprintf("%s (%s)", op.name, op.product),
				Namespace: ns.GetName(),
				Ready:     true,
			}

			v, err := s.kubeClient.GetInstalledOperatorVersion(ctx, types.NamespacedName{
				Name:      op.name,
				Namespace: ns.GetName(),
			})
			if err != nil {
				if errors.Is(err, kubernetes.ErrOperatorNotInstalled) {
					continue // Not installed in this namespace, skip.
				}
				os.Ready = false
				os.Message = fmt.Sprintf("error: %v", err)
				result.Healthy = false
			} else {
				os.Version = fmt.Sprintf("v%s", v.String())
			}

			result.Operators = append(result.Operators, os)
		}
	}
}

func (s *Status) getManagedNamespaces(ctx context.Context, result *OverallStatus) {
	nsList, err := s.kubeClient.GetDBNamespaces(ctx)
	if err != nil {
		s.l.Debugf("Failed to get managed namespaces: %v", err)
		return
	}

	for _, ns := range nsList.Items {
		result.Namespaces = append(result.Namespaces, ns.GetName())
	}
}

func (s *Status) checkDeployment(ctx context.Context, name, namespace string) ComponentStatus {
	cs := ComponentStatus{
		Name:      name,
		Namespace: namespace,
	}

	depl, err := s.kubeClient.GetDeployment(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			cs.Message = "not found"
		} else {
			cs.Message = fmt.Sprintf("error: %v", err)
		}
		return cs
	}

	cs.Ready = isDeploymentReady(depl)
	if !cs.Ready {
		cs.Message = fmt.Sprintf("%d/%d ready",
			depl.Status.ReadyReplicas,
			depl.Status.Replicas,
		)
	}
	return cs
}

func isDeploymentReady(depl *appsv1.Deployment) bool {
	return depl.Status.ReadyReplicas == depl.Status.Replicas &&
		depl.Status.Replicas == depl.Status.UpdatedReplicas &&
		depl.Status.UnavailableReplicas == 0 &&
		depl.GetGeneration() == depl.Status.ObservedGeneration
}

func (s *Status) printJSON(result *OverallStatus) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func (s *Status) printPretty(result *OverallStatus) {
	var out io.Writer = os.Stdout

	// Header
	if result.Healthy {
		_, _ = fmt.Fprint(out, output.Success("Everest %s is healthy", result.EverestVersion))
	} else {
		_, _ = fmt.Fprint(out, output.Failure("Everest %s has issues", result.EverestVersion))
	}

	// Components
	_, _ = fmt.Fprintln(out, "\nComponents:")
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "  NAME\tNAMESPACE\tSTATUS\tMESSAGE")
	for _, c := range result.Components {
		status := "✅ Ready"
		if !c.Ready {
			status = "❌ Not Ready"
		}
		_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", c.Name, c.Namespace, status, c.Message)
	}
	w.Flush()

	// Namespaces
	if len(result.Namespaces) > 0 {
		_, _ = fmt.Fprintf(out, "\nManaged Namespaces: %s\n", strings.Join(result.Namespaces, ", "))
	}

	// Operators
	if len(result.Operators) > 0 {
		_, _ = fmt.Fprintln(out, "\nDatabase Operators:")
		w = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  NAME\tNAMESPACE\tVERSION\tSTATUS\tMESSAGE")
		for _, op := range result.Operators {
			status := "✅ Ready"
			if !op.Ready {
				status = "❌ Not Ready"
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", op.Name, op.Namespace, op.Version, status, op.Message)
		}
		w.Flush()
	}
}
