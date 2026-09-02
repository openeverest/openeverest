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

// Package preflight implements the provider upgrade preflight entrypoint that
// runs inside the provider chart's Helm pre-upgrade hook Job.
//
// The hook Job runs the new provider image in preflight mode before any of
// the release's resources are applied. The target Provider spec is delivered
// by the chart itself — a ConfigMap rendered from the same generated
// provider-spec.yaml the release applies — so the catalog has a single source
// of truth and the provider image needs no embedded definition files.
//
// Design background: spec 009 (provider upgrades) in the openeverest/specs
// repository.
//
// Providers wire it into their main:
//
//	if *preflightSpecFile != "" {
//	    err := preflight.Run(ctx, provider, preflight.Options{TargetSpecFile: *preflightSpecFile})
//	    if err != nil {
//	        os.Exit(1)
//	    }
//	    return
//	}
//
// A non-zero exit aborts the Helm upgrade before anything is applied.
package preflight

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// ErrUpgradeBlocked is returned by Run when the preflight finds at least one
// blocking issue. The hook Job must exit non-zero so the upgrade aborts.
var ErrUpgradeBlocked = errors.New("provider upgrade blocked by preflight")

// Options configures a preflight run.
type Options struct {
	// TargetSpecFile is the path to the target Provider spec — the chart's
	// generated provider-spec.yaml, mounted into the hook Job. The file may
	// be either a bare `spec:` document or a full Provider manifest.
	TargetSpecFile string

	// Out receives the human-readable report. Defaults to os.Stdout.
	Out io.Writer

	// Client overrides the Kubernetes client (used by tests). When nil, a
	// client is built from the ambient kubeconfig / in-cluster config.
	Client client.Client
}

// Run executes the upgrade preflight for provider and writes a report to
// opts.Out. It returns ErrUpgradeBlocked when the upgrade must not proceed,
// or another error when the preflight itself could not run.
func Run(ctx context.Context, provider controller.ProviderInterface, opts Options) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	target, err := loadTargetSpec(opts.TargetSpecFile)
	if err != nil {
		return err
	}

	cl := opts.Client
	if cl == nil {
		if cl, err = newClient(provider); err != nil {
			return err
		}
	}

	current, err := installedProviderSpec(ctx, cl, provider.Name())
	if err != nil {
		return err
	}

	instances, err := providerInstances(ctx, cl, provider.Name())
	if err != nil {
		return err
	}

	c := controller.NewContext(ctx, cl, nil, provider.Name())
	issues := controller.RunUpgradePreflight(c, provider, current, target, instances)

	writeReport(opts.Out, provider.Name(), current, target, len(instances), issues)

	if controller.HasBlockingIssues(issues) {
		return ErrUpgradeBlocked
	}
	return nil
}

// loadTargetSpec reads the target Provider spec from path. Both a bare
// `spec:` document (the generated provider-spec.yaml) and a full Provider
// manifest are accepted.
func loadTargetSpec(path string) (*v1alpha1.ProviderSpec, error) {
	if path == "" {
		return nil, errors.New("preflight: target spec file not set")
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is provided by the chart-controlled hook invocation
	if err != nil {
		return nil, fmt.Errorf("preflight: reading target spec: %w", err)
	}
	var manifest struct {
		Spec v1alpha1.ProviderSpec `json:"spec"`
	}
	if err := yaml.UnmarshalStrict(data, &manifest); err != nil {
		// Full Provider manifests carry apiVersion/kind/metadata; retry
		// non-strict so those fields are tolerated.
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("preflight: parsing target spec: %w", err)
		}
	}
	return &manifest.Spec, nil
}

// newClient builds a direct (uncached) client from the ambient config with
// the core scheme plus the provider's own types, so UpgradeProvider hooks can
// inspect engine resources.
//
//nolint:ireturn // client.Client is the controller-runtime client abstraction
func newClient(provider controller.ProviderInterface) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("preflight: building scheme: %w", err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("preflight: building scheme: %w", err)
	}
	if typesFn := provider.Types(); typesFn != nil {
		if err := typesFn(scheme); err != nil {
			return nil, fmt.Errorf("preflight: registering provider types: %w", err)
		}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("preflight: loading Kubernetes config: %w", err)
	}
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("preflight: creating Kubernetes client: %w", err)
	}
	return cl, nil
}

// installedProviderSpec fetches the currently-installed Provider CR. A
// missing CR is not an error: the hook may run before the provider was ever
// installed, and CheckUpgradePath degrades to a warning.
func installedProviderSpec(ctx context.Context, cl client.Client, name string) (*v1alpha1.ProviderSpec, error) {
	installed := &v1alpha1.Provider{}
	if err := cl.Get(ctx, client.ObjectKey{Name: name}, installed); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil //nolint:nilnil // absence is a valid state, handled by CheckUpgradePath
		}
		return nil, fmt.Errorf("preflight: fetching installed Provider %q: %w", name, err)
	}
	return &installed.Spec, nil
}

// providerInstances lists all Instances managed by the named provider,
// sorted for deterministic reports.
func providerInstances(ctx context.Context, cl client.Client, providerName string) ([]v1alpha1.Instance, error) {
	list := &v1alpha1.InstanceList{}
	if err := cl.List(ctx, list); err != nil {
		return nil, fmt.Errorf("preflight: listing Instances: %w", err)
	}
	instances := make([]v1alpha1.Instance, 0, len(list.Items))
	for _, in := range list.Items {
		if in.Spec.ProviderRef.Name == providerName {
			instances = append(instances, in)
		}
	}
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].Namespace != instances[j].Namespace {
			return instances[i].Namespace < instances[j].Namespace
		}
		return instances[i].Name < instances[j].Name
	})
	return instances, nil
}

func writeReport(w io.Writer, providerName string, current, target *v1alpha1.ProviderSpec, instanceCount int, issues []controller.UpgradeIssue) {
	fmt.Fprintf(w, "Provider %s: %s → %s\n", providerName, releaseVersion(current), releaseVersion(target))
	fmt.Fprintf(w, "Checked %d instance(s).\n\n", instanceCount)

	errCount, warnCount := 0, 0
	for _, issue := range issues {
		marker := "⚠️"
		if issue.Severity == controller.UpgradeError {
			marker = "⛔"
			errCount++
		} else {
			warnCount++
		}
		fmt.Fprintf(w, "%s %s%s\n", marker, issueSubject(issue), issue.Message)
	}
	if len(issues) > 0 {
		fmt.Fprintln(w)
	}

	if errCount > 0 {
		fmt.Fprintf(w, "Result: BLOCKED (%d error(s), %d warning(s)). No changes applied.\n", errCount, warnCount)
		return
	}
	fmt.Fprintf(w, "Result: OK (%d warning(s)). Upgrade may proceed.\n", warnCount)
}

func issueSubject(issue controller.UpgradeIssue) string {
	if issue.InstanceName == "" {
		return ""
	}
	return fmt.Sprintf("[%s/%s] ", issue.Namespace, issue.InstanceName)
}

func releaseVersion(spec *v1alpha1.ProviderSpec) string {
	if spec == nil || spec.Release == nil || spec.Release.Version == "" {
		return "unknown"
	}
	return spec.Release.Version
}
