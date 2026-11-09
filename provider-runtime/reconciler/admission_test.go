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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// admissionTestProvider rejects an even engine replica count, and a version
// change on update.
type admissionTestProvider struct {
	readyTestProvider

	warnings []string
}

func (p *admissionTestProvider) Validate(in controller.ValidationInput) controller.ValidationResult {
	p.calls++
	var errs field.ErrorList
	if r := in.Instance.Spec.Components["engine"].Replicas; r != nil && *r%2 == 0 {
		replicas := field.NewPath("spec", "components").Key("engine").Child("replicas")
		errs = append(errs, field.Invalid(replicas, *r, "must be odd"))
	}
	if in.Old != nil && in.Old.Spec.Version != in.Instance.Spec.Version {
		errs = append(errs, field.Forbidden(field.NewPath("spec", "version"), "may not change"))
	}
	return controller.ValidationResult{Errors: errs, Warnings: p.warnings}
}

func admissionInstance(mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
	in := &v1alpha1.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a"},
		Spec: v1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
			Version:     "7.0",
			Components:  map[string]v1alpha1.ComponentSpec{"engine": {Replicas: new(int32(3))}},
		},
	}
	if mutate != nil {
		mutate(in)
	}
	return in
}

func withReplicas(n int32) func(*v1alpha1.Instance) {
	return func(in *v1alpha1.Instance) {
		in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Replicas: &n}
	}
}

func admissionRequest(t *testing.T, in, old *v1alpha1.Instance) admission.Request {
	t.Helper()
	raw := func(o *v1alpha1.Instance) runtime.RawExtension {
		if o == nil {
			return runtime.RawExtension{}
		}
		data, err := json.Marshal(o)
		require.NoError(t, err)
		return runtime.RawExtension{Raw: data}
	}
	op := admissionv1.Create
	if old != nil {
		op = admissionv1.Update
	}
	return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Operation: op,
		Object:    raw(in),
		OldObject: raw(old),
	}}
}

func newInstanceAdmission(p providerAdapter, objs ...client.Object) *instanceAdmission {
	scheme := newTestScheme()
	_ = corev1.AddToScheme(scheme)
	return &instanceAdmission{
		provider: p,
		client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		decoder:  admission.NewDecoder(scheme),
	}
}

func testProviderCR() *v1alpha1.Provider {
	return &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec: v1alpha1.ProviderSpec{
			Components: map[string]v1alpha1.Component{"engine": {Type: "db"}},
			Versions:   []v1alpha1.VersionBundle{{Name: "7.0"}, {Name: "8.0"}},
		},
	}
}

func causeFields(resp admission.Response) []string {
	if resp.Result == nil || resp.Result.Details == nil {
		return nil
	}
	fields := make([]string, 0, len(resp.Result.Details.Causes))
	for _, c := range resp.Result.Details.Causes {
		fields = append(fields, c.Field)
	}
	return fields
}

func TestInstanceAdmission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          *v1alpha1.Instance
		old         *v1alpha1.Instance
		noProvider  bool
		wantAllowed bool
		wantCauses  []string
		wantCalls   int
	}{
		{
			name:        "valid create",
			in:          admissionInstance(nil),
			wantAllowed: true,
			wantCalls:   1,
		},
		{
			name:       "provider rule",
			in:         admissionInstance(withReplicas(2)),
			wantCauses: []string{"spec.components[engine].replicas"},
			wantCalls:  1,
		},
		{
			name:       "catalog error, before Validate",
			in:         admissionInstance(func(in *v1alpha1.Instance) { in.Spec.Version = "9.0" }),
			wantCauses: []string{"spec.version"},
		},
		{
			name:        "another provider's Instance",
			in:          admissionInstance(func(in *v1alpha1.Instance) { in.Spec.ProviderRef.Name = "other" }),
			wantAllowed: true,
		},
		{
			name: "Instance being deleted",
			in: admissionInstance(func(in *v1alpha1.Instance) {
				now := metav1.Now()
				in.DeletionTimestamp = &now
				in.Finalizers = []string{finalizerName}
				in.Spec.Version = "9.0"
			}),
			wantAllowed: true,
		},
		{
			name:        "missing Provider",
			in:          admissionInstance(withReplicas(2)),
			noProvider:  true,
			wantAllowed: true,
		},
		{
			name:        "update keeps an error the stored Instance had",
			in:          admissionInstance(withReplicas(4)),
			old:         admissionInstance(withReplicas(2)),
			wantAllowed: true,
			wantCalls:   2,
		},
		{
			name:       "update introduces an error",
			in:         admissionInstance(withReplicas(2)),
			old:        admissionInstance(nil),
			wantCauses: []string{"spec.components[engine].replicas"},
			wantCalls:  2,
		},
		{
			name:       "update sees the previous Instance",
			in:         admissionInstance(func(in *v1alpha1.Instance) { in.Spec.Version = "8.0" }),
			old:        admissionInstance(nil),
			wantCauses: []string{"spec.version"},
			wantCalls:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &admissionTestProvider{}
			var objs []client.Object
			if !tc.noProvider {
				objs = append(objs, testProviderCR())
			}
			a := newInstanceAdmission(p, objs...)

			resp := a.Handle(t.Context(), admissionRequest(t, tc.in, tc.old))

			assert.Equal(t, tc.wantAllowed, resp.Allowed)
			assert.Equal(t, tc.wantCalls, p.calls)
			if tc.wantAllowed {
				return
			}
			require.NotNil(t, resp.Result)
			assert.Equal(t, int32(http.StatusUnprocessableEntity), resp.Result.Code)
			assert.Equal(t, metav1.StatusReasonInvalid, resp.Result.Reason)
			assert.Equal(t, tc.wantCauses, causeFields(resp))
		})
	}
}

func TestInstanceAdmissionWarnings(t *testing.T) {
	t.Parallel()
	p := &admissionTestProvider{warnings: []string{"spec.components[engine].version: version 6.0 is deprecated"}}
	a := newInstanceAdmission(p, testProviderCR())

	resp := a.Handle(t.Context(), admissionRequest(t, admissionInstance(nil), nil))
	assert.True(t, resp.Allowed)
	assert.Equal(t, p.warnings, resp.Warnings)

	resp = a.Handle(t.Context(), admissionRequest(t, admissionInstance(withReplicas(2)), nil))
	assert.False(t, resp.Allowed)
	assert.Equal(t, p.warnings, resp.Warnings, "a denial carries the warnings too")
}

// admissionBackupProvider is admissionTestProvider with backups.
type admissionBackupProvider struct {
	admissionTestProvider
}

func (p *admissionBackupProvider) SyncBackup(*controller.Context, *backupv1alpha1.Backup) (controller.BackupExecutionStatus, error) {
	return controller.BackupExecutionStatus{}, nil
}

func (p *admissionBackupProvider) SyncRestore(*controller.Context, *backupv1alpha1.Restore) (controller.RestoreExecutionStatus, error) {
	return controller.RestoreExecutionStatus{}, nil
}

func (p *admissionBackupProvider) CleanupBackup(*controller.Context, *backupv1alpha1.Backup) (bool, error) {
	return true, nil
}

func (p *admissionBackupProvider) CleanupRestore(*controller.Context, *backupv1alpha1.Restore) (bool, error) {
	return true, nil
}

func TestInstanceAdmissionBackupClass(t *testing.T) {
	t.Parallel()
	limited := func() *backupv1alpha1.BackupClass {
		bc := compatibleBackupClass()
		bc.Spec.ProviderManaged.Limits = &backupv1alpha1.BackupClassLimits{MaxStorages: new(int32(1))}
		return bc
	}
	ours := func(c backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupClass {
		return judged(testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"), "test-provider", c, reason)
	}
	otherProvider := func() *backupv1alpha1.BackupClass {
		return judged(testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "other"), "other", backupv1alpha1.CompatibilityCompatible, "")
	}
	stale := func() *backupv1alpha1.BackupClass {
		bc := compatibleBackupClass()
		bc.Generation = 2
		return bc
	}
	storages := func() []client.Object {
		return []client.Object{
			testBackupStorage("s3-a", "keys", metav1.ConditionTrue), testBackupStorage("s3-b", "keys", metav1.ConditionTrue), testSecret("keys"),
		}
	}

	tests := []struct {
		name         string
		in           *v1alpha1.Instance
		old          *v1alpha1.Instance
		class        *backupv1alpha1.BackupClass
		wantAllowed  bool
		wantCauses   []string
		wantWarnings []string
	}{
		{
			name:        "compatible class",
			in:          admissionInstance(withBackup(true, "s3-a")),
			class:       limited(),
			wantAllowed: true,
		},
		{
			// Only the published verdict decides: a mode this runtime does
			// not know is no reason to reject (spec 019 §4.8 invariant 1).
			name:        "compatible class of an execution mode the runtime does not know",
			in:          admissionInstance(withBackup(true, "s3-a")),
			class:       judged(testBackupClass(otherMode), "test-provider", backupv1alpha1.CompatibilityCompatible, ""),
			wantAllowed: true,
		},
		{
			name:        "pending class",
			in:          admissionInstance(withBackup(true, "s3-a")),
			class:       ours(backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted),
			wantAllowed: true,
			wantWarnings: []string{
				"backup class pbm is pending for provider test-provider, which holds the instance until it is compatible: ClassNotAccepted",
			},
		},
		{
			name:        "class not judged yet",
			in:          admissionInstance(withBackup(true, "s3-a")),
			class:       testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"),
			wantAllowed: true,
			wantWarnings: []string{
				"backup class pbm is pending for provider test-provider, which holds the instance until it is compatible: " +
					`VerdictPending: BackupClass "pbm" has not been judged for Provider "test-provider" at generation 1 yet`,
			},
		},
		{
			name:        "verdict for an older generation",
			in:          admissionInstance(withBackup(true, "s3-a")),
			class:       stale(),
			wantAllowed: true,
			wantWarnings: []string{
				"backup class pbm is pending for provider test-provider, which holds the instance until it is compatible: " +
					`VerdictPending: the verdict on BackupClass "pbm" for Provider "test-provider" predates generation 2 of the class`,
			},
		},
		{
			name:       "incompatible class",
			in:         admissionInstance(withBackup(true, "s3-a")),
			class:      ours(backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift),
			wantCauses: []string{"spec.backup.classRef.name"},
		},
		{
			name:       "class of another provider",
			in:         admissionInstance(withBackup(true, "s3-a")),
			class:      otherProvider(),
			wantCauses: []string{"spec.backup.classRef.name"},
		},
		{
			name:       "more storages than the class allows",
			in:         admissionInstance(withBackup(true, "s3-a", "s3-b")),
			class:      limited(),
			wantCauses: []string{"spec.backup.storages"},
		},
		{
			name:         "missing class",
			in:           admissionInstance(withBackup(true, "s3-a")),
			wantAllowed:  true,
			wantWarnings: []string{"backup class pbm not found"},
		},
		{
			name:        "backups off",
			in:          admissionInstance(withBackup(false, "s3-a")),
			class:       otherProvider(),
			wantAllowed: true,
		},
		{
			name:        "update keeps an incompatibility the stored Instance had",
			in:          admissionInstance(func(in *v1alpha1.Instance) { withBackup(true, "s3-a")(in); withReplicas(5)(in) }),
			old:         admissionInstance(withBackup(true, "s3-a")),
			class:       otherProvider(),
			wantAllowed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			objs := append(storages(), testProviderCR())
			if tc.class != nil {
				objs = append(objs, tc.class)
			}
			a := newInstanceAdmission(&admissionBackupProvider{}, objs...)

			resp := a.Handle(t.Context(), admissionRequest(t, tc.in, tc.old))

			assert.Equal(t, tc.wantAllowed, resp.Allowed)
			assert.Equal(t, tc.wantCauses, causeFields(resp))
			assert.Equal(t, tc.wantWarnings, resp.Warnings)
		})
	}
}

func TestInstanceAdmissionIncompatibleClassMessage(t *testing.T) {
	t.Parallel()
	bc := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
	bc.Status.Providers = []backupv1alpha1.BackupClassProviderCompatibility{{
		Name: "test-provider", Compatibility: backupv1alpha1.CompatibilityIncompatible, ClassGeneration: 1,
		Reason: backupv1alpha1.ReasonSpecDrift, Message: "spec.supportsImport differs",
	}}
	a := newInstanceAdmission(&admissionBackupProvider{}, testProviderCR(), bc)

	resp := a.Handle(t.Context(), admissionRequest(t, admissionInstance(withBackup(true)), nil))

	assert.False(t, resp.Allowed)
	require.NotNil(t, resp.Result)
	require.NotNil(t, resp.Result.Details)
	require.Len(t, resp.Result.Details.Causes, 1)
	assert.Equal(t,
		`Invalid value: "pbm": is incompatible with provider test-provider: SpecDrift: spec.supportsImport differs`,
		resp.Result.Details.Causes[0].Message,
	)
}

// admissionReferenceProvider is admissionTestProvider with ValidateReferences.
type admissionReferenceProvider struct {
	admissionTestProvider

	referenceErrs field.ErrorList
	referencesErr error
}

func (p *admissionReferenceProvider) ValidateReferences(*controller.Context) (field.ErrorList, error) {
	return p.referenceErrs, p.referencesErr
}

func TestInstanceAdmissionReferenceWarnings(t *testing.T) {
	t.Parallel()
	userSecret := field.NewPath("spec", "userSecretRef", "name")
	withUserSecret := func(in *v1alpha1.Instance) { in.Spec.UserSecretRef = &common.SecretRef{Name: "orders-users"} }

	tests := []struct {
		name         string
		provider     *admissionReferenceProvider
		objs         []client.Object
		wantWarnings []string
	}{
		{
			name:         "missing user Secret",
			provider:     &admissionReferenceProvider{},
			wantWarnings: []string{"secret orders-users named by spec.userSecretRef not found in team-a"},
		},
		{
			name: "contents the provider rejects",
			provider: &admissionReferenceProvider{referenceErrs: field.ErrorList{
				field.Invalid(userSecret, "orders-users", `missing "MONGODB_BACKUP_USER"`),
			}},
			objs:         []client.Object{testSecret("orders-users")},
			wantWarnings: []string{`spec.userSecretRef.name: Invalid value: "orders-users": missing "MONGODB_BACKUP_USER"`},
		},
		{
			name:     "a check that could not run",
			provider: &admissionReferenceProvider{referencesErr: errors.New("connection refused")},
			objs:     []client.Object{testSecret("orders-users")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := newInstanceAdmission(tc.provider, append(tc.objs, testProviderCR())...)

			resp := a.Handle(t.Context(), admissionRequest(t, admissionInstance(withUserSecret), nil))

			assert.True(t, resp.Allowed, "admission never rejects on the contents of another object")
			assert.Equal(t, tc.wantWarnings, resp.Warnings)
		})
	}
}

func TestCheckAdmissionCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.ErrorContains(t, checkAdmissionCert(dir), "serving certificate is missing")

	for _, name := range []string{"tls.crt", "tls.key"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}
	require.NoError(t, checkAdmissionCert(dir))
}

// webhookTestProvider has a name of its own, as controller names are unique
// per process.
type webhookTestProvider struct {
	readinessTestProvider
}

func (p *webhookTestProvider) Name() string { return "webhook-test" }

func TestWithAdmissionRegistersTheWebhook(t *testing.T) {
	dir := t.TempDir()
	kc := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kc, []byte(unreachableKubeconfig), 0o600))
	t.Setenv("KUBECONFIG", kc)
	certDir := filepath.Join(dir, "certs")
	require.NoError(t, os.Mkdir(certDir, 0o700))
	withCertDir := func(o *reconcilerOptions) { o.admissionCertDir = certDir }

	_, err := New(t.Context(), &webhookTestProvider{}, WithMetrics("0"), withCertDir)
	require.ErrorContains(t, err, "serving certificate is missing")

	for _, name := range []string{"tls.crt", "tls.key"} {
		require.NoError(t, os.WriteFile(filepath.Join(certDir, name), []byte("x"), 0o600))
	}
	r, err := New(t.Context(), &webhookTestProvider{}, WithMetrics("0"), withCertDir)
	require.NoError(t, err)

	mux := r.GetManager().GetWebhookServer().WebhookMux()
	_, pattern := mux.Handler(httptest.NewRequestWithContext(t.Context(), http.MethodPost, AdmissionPath, nil))
	assert.Equal(t, AdmissionPath, pattern)
}
