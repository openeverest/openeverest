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

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func TestParseUninstallCheckFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		args        []string
		wantTimeout time.Duration
		wantErr     error
		wantAnyErr  bool
	}{
		{name: "defaults", wantTimeout: time.Minute},
		{name: "a timeout and a log level", args: []string{"--timeout=30s", "--zap-log-level=debug"}, wantTimeout: 30 * time.Second},
		{name: "help", args: []string{"--help"}, wantErr: flag.ErrHelp},
		{name: "an unknown flag", args: []string{"--scale-controller"}, wantAnyErr: true},
		{name: "a stray argument", args: []string{"now"}, wantAnyErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := parseUninstallCheckFlags(tc.args, io.Discard)

			switch {
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			case tc.wantAnyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tc.wantTimeout, cfg.timeout)
			}
		})
	}
}

func TestUninstallCheck(t *testing.T) {
	t.Parallel()

	held := &monitoringv1alpha1.MonitoringDestination{ObjectMeta: metav1.ObjectMeta{
		Namespace: "team-a", Name: "prometheus", Finalizers: []string{monitoringv1alpha1.InUseProtectionFinalizer},
	}}
	forbidden := interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "", errors.New("no"))
		},
	}
	testCases := []struct {
		name       string
		objs       []client.Object
		funcs      interceptor.Funcs
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "nothing held",
			wantCode:   exitNothingHeld,
			wantStdout: "OpenEverest protects no object any more; it can be uninstalled.\n",
		},
		{
			name:       "a destination held",
			objs:       []client.Object{held},
			wantCode:   exitHeld,
			wantStderr: "1 MonitoringDestination: team-a/prometheus\n",
		},
		{
			name:       "the check fails",
			funcs:      forbidden,
			wantCode:   exitError,
			wantStderr: "is forbidden",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objs...).WithInterceptorFuncs(tc.funcs).Build()
			var stdout, stderr bytes.Buffer

			code := uninstallCheck(t.Context(), c, &stdout, &stderr)

			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantStdout, stdout.String())
			if tc.wantStderr == "" {
				assert.Empty(t, stderr.String())
			} else {
				assert.Contains(t, stderr.String(), tc.wantStderr)
			}
		})
	}
}
