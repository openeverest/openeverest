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
	"net/http"
	"os"
	"path/filepath"

	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/validation"
)

const (
	// AdmissionPath is where the Instance admission webhook is served.
	AdmissionPath = "/admission/instances"
	// AdmissionPort is the port of the webhook server.
	AdmissionPort = 9443
	// AdmissionCertDir holds the serving certificate, as tls.crt and tls.key.
	AdmissionCertDir = "/tmp/k8s-webhook-server/serving-certs"
)

// WithAdmission serves the Instance admission webhook at AdmissionPath, on
// port AdmissionPort, with the certificate mounted in AdmissionCertDir. It
// runs the checks the reconciler runs before Sync, on every replica, so a
// write is rejected before it is persisted. The provider chart registers the
// webhook with failurePolicy Ignore, so the reconciler stays authoritative
// when the provider is down.
func WithAdmission() ReconcilerOption {
	return func(o *reconcilerOptions) {
		o.admissionCertDir = AdmissionCertDir
	}
}

// checkAdmissionCert fails fast when the serving certificate is not mounted.
func checkAdmissionCert(dir string) error {
	for _, name := range []string{"tls.crt", "tls.key"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf(
				"admission is enabled but its serving certificate is missing; "+
					"mount the Secret from the provider chart's admission template at %s: %w",
				dir, err,
			)
		}
	}
	return nil
}

// instanceAdmission validates writes to this provider's Instances.
type instanceAdmission struct {
	provider providerAdapter
	reader   client.Reader
	decoder  admission.Decoder
}

// Handle admits a write when the Instance passes the validation pipeline. An
// error the stored Instance already had does not block an update, so an
// Instance written under an older provider release stays editable.
func (a *instanceAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {
	in := &v1alpha1.Instance{}
	if err := a.decoder.DecodeRaw(req.Object, in); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if in.Spec.ProviderRef.Name != a.provider.Name() || !in.DeletionTimestamp.IsZero() {
		return admission.Allowed("")
	}

	provider := &v1alpha1.Provider{}
	if err := a.reader.Get(ctx, client.ObjectKey{Name: in.Spec.ProviderRef.Name}, provider); err != nil {
		// The reconciler reports a missing Provider; admission never rejects a missing referent.
		log.FromContext(ctx).Info("Admitting an Instance without validation", "reason", err.Error())
		return admission.Allowed("")
	}

	var old *v1alpha1.Instance
	if req.Operation == admissionv1.Update && len(req.OldObject.Raw) > 0 {
		old = &v1alpha1.Instance{}
		if err := a.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	}

	res := validation.Run(ctx, a.provider, controller.ValidationInput{Instance: in, Old: old, Provider: provider})
	errs := res.Errors
	if old != nil && len(errs) > 0 {
		previous := validation.Run(ctx, a.provider, controller.ValidationInput{Instance: old, Provider: provider})
		errs = ratchet(errs, previous.Errors)
	}
	if len(errs) == 0 {
		return admission.Allowed("").WithWarnings(res.Warnings...)
	}

	status := apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("Instance").GroupKind(), in.Name, errs).ErrStatus
	return admission.Response{AdmissionResponse: admissionv1.AdmissionResponse{
		Allowed:  false,
		Result:   &status,
		Warnings: res.Warnings,
	}}
}

// ratchet drops every error whose type and field path the previous object
// already had.
func ratchet(errs, previous field.ErrorList) field.ErrorList {
	type key struct {
		typ   field.ErrorType
		field string
	}
	had := make(map[key]bool, len(previous))
	for _, err := range previous {
		had[key{err.Type, err.Field}] = true
	}
	var kept field.ErrorList
	for _, err := range errs {
		if !had[key{err.Type, err.Field}] {
			kept = append(kept, err)
		}
	}
	return kept
}
