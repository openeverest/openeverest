// everest
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

package kubernetes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

type receivedWarning struct {
	code int
	text string
}

type warningRecorder struct {
	warnings []receivedWarning
}

func (r *warningRecorder) HandleWarningHeaderWithContext(_ context.Context, code int, _, text string) {
	r.warnings = append(r.warnings, receivedWarning{code: code, text: text})
}

func TestCollectingWarningHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		collect   bool
		warnings  []receivedWarning
		collected []string
	}{
		{
			name:    "distinct texts in arrival order",
			collect: true,
			warnings: []receivedWarning{
				{code: 299, text: "secret db-creds not found"},
				{code: 299, text: "backup class nightly not found"},
				{code: 299, text: "secret db-creds not found"},
			},
			collected: []string{"secret db-creds not found", "backup class nightly not found"},
		},
		{
			name:    "only non-empty 299 warnings",
			collect: true,
			warnings: []receivedWarning{
				{code: 199, text: "miscellaneous warning"},
				{code: 299, text: ""},
			},
		},
		{
			name: "no collector",
			warnings: []receivedWarning{
				{code: 299, text: "secret db-creds not found"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			var collector *WarningCollector
			if tt.collect {
				ctx, collector = WithWarningCollector(ctx)
			}
			next := &warningRecorder{}
			h := CollectingWarningHandler{Next: next}
			for _, w := range tt.warnings {
				h.HandleWarningHeaderWithContext(ctx, w.code, "-", w.text)
			}

			assert.Equal(t, tt.warnings, next.warnings, "every warning reaches Next")
			if collector != nil {
				assert.Equal(t, tt.collected, collector.Warnings())
			}
		})
	}
}

// The client hands the warnings of a write to the collector of the context the write was made with.
func TestCollectingWarningHandler_ClientWrite(t *testing.T) {
	t.Parallel()

	invalid := k8serrors.NewInvalid(
		v1alpha1.GroupVersion.WithKind("Instance").GroupKind(), "my-db",
		field.ErrorList{field.Required(field.NewPath("spec", "topology", "type"), "")},
	).ErrStatus
	invalid.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}

	tests := []struct {
		name        string
		status      int
		body        any
		wantInvalid bool
	}{
		{
			name:   "admitted",
			status: http.StatusCreated,
			body: &v1alpha1.Instance{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "Instance"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "everest"},
			},
		},
		{
			name:        "denied",
			status:      http.StatusUnprocessableEntity,
			body:        &invalid,
			wantInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/apis/core.openeverest.io/v1alpha1/namespaces/everest/instances", r.URL.Path)
				w.Header().Add("Warning", `299 - "secret \"db-creds\" named by spec.userSecretRef not found"`)
				w.Header().Add("Warning", `299 - "backup class nightly not found"`)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				assert.NoError(t, json.NewEncoder(w).Encode(tt.body))
			}))
			defer srv.Close()

			mapper := meta.NewDefaultRESTMapper(nil)
			mapper.Add(v1alpha1.GroupVersion.WithKind("Instance"), meta.RESTScopeNamespace)
			k8sClient, err := ctrlclient.New(
				&rest.Config{
					Host:                      srv.URL,
					WarningHandlerWithContext: CollectingWarningHandler{Next: rest.NoWarnings{}},
				},
				ctrlclient.Options{Scheme: CreateScheme(), Mapper: mapper},
			)
			require.NoError(t, err)

			ctx, collector := WithWarningCollector(context.Background())
			err = k8sClient.Create(ctx, &v1alpha1.Instance{
				ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "everest"},
			})
			if tt.wantInvalid {
				assert.True(t, k8serrors.IsInvalid(err), "got %v", err)
			} else {
				require.NoError(t, err)
			}
			want := []string{
				`secret "db-creds" named by spec.userSecretRef not found`,
				"backup class nightly not found",
			}
			assert.Equal(t, want, collector.Warnings())
		})
	}
}

// A warning the caller raises itself is reported once with those the Kubernetes API returns.
func TestAddWarning(t *testing.T) {
	t.Parallel()

	AddWarning(context.Background(), "no collector, no effect")

	ctx, collector := WithWarningCollector(context.Background())
	AddWarning(ctx, `BackupClass "pbm" is Pending for Provider "psmdb"`)
	AddWarning(ctx, "")
	CollectingWarningHandler{Next: &warningRecorder{}}.HandleWarningHeaderWithContext(
		ctx, 299, "-", `BackupClass "pbm" is Pending for Provider "psmdb"`,
	)

	assert.Equal(t, []string{`BackupClass "pbm" is Pending for Provider "psmdb"`}, collector.Warnings())
}
