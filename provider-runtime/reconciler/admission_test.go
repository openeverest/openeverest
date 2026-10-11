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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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
	return &instanceAdmission{
		provider: p,
		reader:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
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
