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
	"slices"

	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/internal/backupclass"
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
	client   client.Client
	decoder  admission.Decoder
}

// Handle admits a write when the Instance passes the validation pipeline and
// fits the BackupClass it names. An error the stored Instance already had does
// not block an update, so an Instance written under an older provider release
// stays editable. Problems with the other objects the Instance names come back
// as warnings: those objects may be created after the Instance.
func (a *instanceAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {
	in := &v1alpha1.Instance{}
	if err := a.decoder.DecodeRaw(req.Object, in); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if in.Spec.ProviderRef.Name != a.provider.Name() || !in.DeletionTimestamp.IsZero() {
		return admission.Allowed("")
	}
	logger := log.FromContext(ctx)

	provider := &v1alpha1.Provider{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: in.Spec.ProviderRef.Name}, provider); err != nil {
		logger.Info("Admitting an Instance without validation", "reason", err.Error())
		if apierrors.IsNotFound(err) {
			return admission.Allowed("").WithWarnings(fmt.Sprintf("provider %s not found", in.Spec.ProviderRef.Name))
		}
		return admission.Allowed("")
	}

	var old *v1alpha1.Instance
	if req.Operation == admissionv1.Update && len(req.OldObject.Raw) > 0 {
		old = &v1alpha1.Instance{}
		if err := a.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	}

	errs, warnings, err := a.check(ctx, in, old, provider)
	if err == nil && old != nil && len(errs) > 0 {
		var previous field.ErrorList
		previous, _, err = a.check(ctx, old, nil, provider)
		errs = ratchet(errs, previous)
	}
	if err != nil {
		logger.Info("Admitting an Instance without validation", "reason", err.Error())
		return admission.Allowed("")
	}
	if len(errs) == 0 {
		return admission.Allowed("").WithWarnings(append(warnings, a.referenceWarnings(ctx, in)...)...)
	}

	status := apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("Instance").GroupKind(), in.Name, errs).ErrStatus
	return admission.Response{AdmissionResponse: admissionv1.AdmissionResponse{
		Allowed:  false,
		Result:   &status,
		Warnings: warnings,
	}}
}

// check runs the validation pipeline on in and checks it against the
// BackupClass it names, reading the class's compatibility with the provider
// from its status.providers: Incompatible is an error, a missing class or
// one that is Pending a warning.
func (a *instanceAdmission) check(
	ctx context.Context, in, old *v1alpha1.Instance, provider *v1alpha1.Provider,
) (field.ErrorList, []string, error) {
	res := validation.Run(ctx, a.provider, controller.ValidationInput{Instance: in, Old: old, Provider: provider})
	errs, warnings := slices.Clip(res.Errors), slices.Clip(res.Warnings)
	if _, ok := a.provider.(controller.BackupProvider); !ok || in.Spec.Backup == nil || !in.Spec.Backup.Enabled {
		return errs, warnings, nil
	}

	name := in.Spec.Backup.ClassRef.Name
	class := &backupv1alpha1.BackupClass{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: name}, class); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, fmt.Errorf("get backup class %s: %w", name, err)
		}
		return errs, append(warnings, fmt.Sprintf("backup class %s not found", name)), nil
	}
	v := backupclass.ProviderVerdict(class, a.provider.Name())
	if v.Compatibility == backupv1alpha1.CompatibilityIncompatible {
		return append(errs, field.Invalid(field.NewPath("spec", "backup", "classRef", "name"), name,
			fmt.Sprintf("is incompatible with provider %s: %s", a.provider.Name(), backupclass.VerdictDetail(v)),
		)), warnings, nil
	}
	if v.Compatibility == backupv1alpha1.CompatibilityPending {
		warnings = append(warnings, fmt.Sprintf("backup class %s is pending for provider %s, which holds the instance until it is compatible: %s",
			name, a.provider.Name(), backupclass.VerdictDetail(v),
		))
	}
	errs = append(errs, backupclass.LimitErrors(in, class)...)
	return append(errs, backupclass.PITRErrors(in, class)...), warnings, nil
}

// referenceWarnings reports what the reconciler would find wrong with the
// objects in names: a missing one, or one whose contents the provider's
// ValidateReferences rejects.
func (a *instanceAdmission) referenceWarnings(ctx context.Context, in *v1alpha1.Instance) []string {
	logger := log.FromContext(ctx)
	var missing []string
	b, err := missingReferent(ctx, a.client, in)
	if err != nil {
		logger.Info("Skipping the reference checks", "reason", err.Error())
		return nil
	}
	if b.reason != "" {
		missing = append(missing, b.message)
	}
	if _, ok := a.provider.(controller.BackupProvider); ok && in.Spec.Backup != nil && in.Spec.Backup.Enabled {
		for _, name := range backupStorageNames(in) {
			b, err := missingBackupStorage(ctx, a.client, in.Namespace, name)
			if err != nil {
				logger.Info("Skipping the reference checks", "reason", err.Error())
				return missing
			}
			if b.reason != "" {
				missing = append(missing, b.message)
			}
		}
	}
	rv, ok := a.provider.(controller.ReferenceValidator)
	if len(missing) > 0 || !ok {
		return missing
	}

	errs, err := rv.ValidateReferences(newContext(ctx, a.client, in, a.provider, a.provider.Name()))
	if err != nil {
		logger.Info("Skipping the reference checks", "reason", err.Error())
		return nil
	}
	warnings := make([]string, 0, len(errs))
	for _, err := range errs {
		warnings = append(warnings, err.Error())
	}
	return warnings
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
