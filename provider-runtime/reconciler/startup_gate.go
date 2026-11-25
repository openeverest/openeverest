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

package reconciler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/gate"
)

// The startup gate reads the CRD core publishes its window on and the
// provider's own Provider, writes the contract version that starts on the
// Provider's status, and records Events on the Provider. Besides the base
// rules every provider declares, it needs these, with the provider's name in
// place of <provider>, in the provider's own RBAC markers (controller-gen
// does not read the markers of a module's dependencies):
//
//	// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,resourceNames=providers.core.openeverest.io,verbs=get
//	// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers/status,resourceNames=<provider>,verbs=patch
//	// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// providerCRD is the CRD of the Provider kind, on which core publishes the
// window the startup gate checks the provider against.
const providerCRD = "providers.core.openeverest.io"

// startupGateAction is the action of the Events the startup gate records.
const startupGateAction = "Start"

// startupGateOptions configures the startup gate. Only tests set them, with
// withStartupGate: the gate cannot be turned off in production.
type startupGateOptions struct {
	// skip turns the gate off, for tests of what runs after it.
	skip bool
	// interval is how often a held build reads again; the gate's default
	// when zero.
	interval time.Duration
	// linked replaces the contract version the build links.
	linked string
}

// withStartupGate configures the startup gate.
func withStartupGate(o startupGateOptions) ReconcilerOption {
	return func(opts *reconcilerOptions) {
		opts.startupGate = o
	}
}

// awaitCompatibleCore runs the startup gate. It returns true once this build
// may start, and then records its contract version on the Provider in the
// background, or false once ctx is done.
//
// It runs before the manager starts, as the manager starts the controllers,
// the admission webhook server and the metrics server: a build OpenEverest
// does not support must neither reconcile nor admit anything. The health
// server is already up, so the pod stays alive, and unready, while it waits:
// /readyz reports ready only once the manager's caches have synced. Under
// the RollingUpdate of the SDK charts the pods of the previous build keep
// serving meanwhile, and this one starts by itself once OpenEverest is
// upgraded.
func (r *ProviderReconciler) awaitCompatibleCore(ctx context.Context) bool {
	if r.startupGate.skip {
		return true
	}
	logger := ctrl.Log.WithName("startup-gate")
	var provider *v1alpha1.Provider
	held := false
	runner := r.startupGateRunner(func(p *v1alpha1.Provider) { provider = p })
	runner.Held = func(_ context.Context, d gate.Decision) {
		held = true
		logger.Info("Waiting for an OpenEverest that supports this build", "reason", d.Reason, "waitingFor", d.WaitingFor, "message", d.Message)
		r.recordGateEvent(provider, corev1.EventTypeWarning, d)
	}
	runner.WriteFailed = func(_ context.Context, err error) {
		if apierrors.IsNotFound(err) {
			logger.Info("The Provider does not exist yet; recording the contract version this build links once it does")
			return
		}
		logger.Error(err, "Cannot record the contract version this build links on its Provider; trying again")
	}

	d, err := runner.Wait(ctx)
	if err != nil {
		return false
	}
	logger.Info("Starting", "reason", d.Reason, "message", d.Message)
	switch {
	case d.Acknowledged:
		r.recordGateEvent(provider, corev1.EventTypeWarning, d)
	case held:
		r.recordGateEvent(provider, corev1.EventTypeNormal, d)
	}
	go func() { _ = runner.Record(ctx, d) }()
	return true
}

// startupGateRunner returns the gate of this provider, reading through the
// API reader, as the manager's caches do not run yet and a held build must
// see a core upgrade as soon as it happens. read is told about every
// Provider it reads.
func (r *ProviderReconciler) startupGateRunner(read func(*v1alpha1.Provider)) gate.Runner {
	name := r.provider.Name()
	return gate.Runner{
		Name:     name,
		Linked:   r.startupGate.linked,
		Interval: r.startupGate.interval,
		ReadRoot: func(ctx context.Context) (gate.Root, error) {
			p := &v1alpha1.Provider{}
			if err := r.apiReader.Get(ctx, client.ObjectKey{Name: name}, p); err != nil {
				if apierrors.IsNotFound(err) {
					return gate.Root{}, nil
				}
				return gate.Root{}, fmt.Errorf("get provider %s: %w", name, err)
			}
			read(p)
			return gate.Root{Observed: p.Status.ContractVersion, Acknowledged: p.Annotations[contract.AcknowledgedAnnotation]}, nil
		},
		ReadCore: func(ctx context.Context) (gate.Core, error) {
			crd := &metav1.PartialObjectMetadata{}
			crd.SetGroupVersionKind(apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
			if err := r.apiReader.Get(ctx, client.ObjectKey{Name: providerCRD}, crd); err != nil {
				if apierrors.IsNotFound(err) {
					return gate.Core{}, nil
				}
				return gate.Core{}, fmt.Errorf("get CustomResourceDefinition %s: %w", providerCRD, err)
			}
			return gate.CoreFromCRD(crd.Labels, crd.Annotations), nil
		},
		WriteObserved: r.applyContractVersion,
	}
}

// applyContractVersion server-side-applies version as the contract version
// observed on the provider's Provider, under the provider's status field
// owner. An apply owns only the fields it sets, so it leaves the status
// fields other writers own alone, and on the status subresource it fails
// rather than create a Provider that does not exist yet.
func (r *ProviderReconciler) applyContractVersion(ctx context.Context, version string) error {
	name := r.provider.Name()
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": v1alpha1.GroupVersion.String(),
		"kind":       providerKind,
		"metadata":   map[string]any{"name": name},
		"status":     map[string]any{"contractVersion": version},
	}}
	if err := r.Status().Apply(ctx, client.ApplyConfigurationFromUnstructured(u),
		client.FieldOwner(contract.ProviderStatusFieldOwner(name)), client.ForceOwnership,
	); err != nil {
		return fmt.Errorf("record contract version %s on provider %s: %w", version, name, err)
	}
	return nil
}

// providerKind is the kind of Provider objects.
const providerKind = "Provider"

// recordGateEvent records d as an Event on provider, the Provider last
// read, so that kubectl describe shows it; on a Provider that does not
// exist yet it records it by name.
func (r *ProviderReconciler) recordGateEvent(provider *v1alpha1.Provider, eventType string, d gate.Decision) {
	if r.recorder == nil {
		return
	}
	if provider == nil {
		provider = &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: r.provider.Name()}}
	}
	r.recorder.Eventf(provider, nil, eventType, d.Reason, startupGateAction, "%s", d.Message)
}
