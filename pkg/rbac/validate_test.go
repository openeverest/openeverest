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

package rbac

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePolicy(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		path string
		err  error
	}{
		{
			path: "./testdata/policy-1-good.csv",
			err:  nil,
		},
		{
			path: "./testdata/policy-2-bad.csv",
			err:  errPolicySyntax,
		},
		{
			path: "./testdata/policy-3-bad.csv",
			err:  errPolicySyntax,
		},
		{
			path: "./testdata/policy-4-bad.csv",
			err:  errPolicySyntax,
		},
		{
			path: "./testdata/policy-5-bad.csv",
			err:  errPolicySyntax,
		},
		{
			path: "./testdata/policy-6-bad.csv",
			err:  errPolicySyntax,
		},
		{
			path: "./testdata/policy-7-bad.csv",
			err:  errPolicySyntax,
		},
	}

	ctx := context.Background()
	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			t.Parallel()
			err := ValidatePolicy(ctx, nil, tc.path)
			if err != nil && tc.err == nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if err == nil && tc.err != nil {
				t.Fatalf("expected error %v, got nil", tc.err)
			}
			if !errors.Is(err, tc.err) {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
}

func TestCheckResourceNames(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		policies [][]string
		valid    bool
	}{
		{
			policies: [][]string{
				{"role:admin", "instances", "create", "*"},
				{"role:admin", "monitoring-configs", "*", "*"},
			},
			valid: true,
		},
		{
			policies: [][]string{
				{"role:admin", "instances", "create", "*"},
				{"role:admin", "monitoring-configs", "*", "*"},
				{"role:admin", "does-not-exist", "*", "*"},
			},
			valid: false,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			t.Parallel()
			err := checkResourceNames(tc.policies)
			if err != nil && tc.valid {
				t.Fatalf("expected no error, got %v", err)
			}
			if err == nil && !tc.valid {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestCheckRoles(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		roles    []string
		policies [][]string
		valid    bool
	}{
		{
			roles: []string{"role:admin", "role:viewer"},
			policies: [][]string{
				{"role:admin", "instances", "create", "*"},
				{"role:admin", "monitoring-configs", "*", "*"},
			},
			valid: true,
		},
		{
			roles: []string{"role:admin", "role:viewer"},
			policies: [][]string{
				{"role:admin", "instances", "create", "*"},
				{"role:admin", "monitoring-configs", "*", "*"},
				{"role:does-not-exist", "monitoring-configs", "*", "*"},
			},
			valid: false,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			t.Parallel()
			err := checkRoles(tc.roles, tc.policies)
			if err != nil && tc.valid {
				t.Fatalf("expected no error, got %v", err)
			}
			if err == nil && !tc.valid {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestValidateTerms(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		terms []string
		valid bool
	}{
		{
			terms: []string{"role:admin", "instances", "create", "*"},
			valid: true,
		},
		{
			terms: []string{"role:admin!!", "instances", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin!!", "instances names", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin!!", "", "create", "*"},
			valid: false,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			t.Parallel()
			err := validateTerms(tc.terms)
			if err != nil && tc.valid {
				t.Fatalf("expected no error, got %v", err)
			}
			if err == nil && !tc.valid {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestCan(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		request []string
		can     bool
	}{
		{
			request: []string{
				"admin",
				"create",
				"instances",
				"prod/ns1/test-cluster",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"read",
				"instances",
				"prod/ns1/test-cluster",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"update",
				"instances",
				"prod/ns1/test-cluster",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"update",
				"backups",
				"prod/ns1/test-backup",
			},
			can: true,
		},
		{
			request: []string{
				"alice",
				"create",
				"instances",
				"dev/ns1/test",
			},
			can: false,
		},
		{
			request: []string{
				"alice",
				"read",
				"providers",
				"prod/psmdb",
			},
			can: true,
		},
		{
			request: []string{
				"alice",
				"create",
				"instances",
				"prod/ns1/alice-cluster-1",
			},
			can: true,
		},
		{
			request: []string{
				"bob",
				"create",
				"instances",
				"prod/ns1/test",
			},
			can: false,
		},
		{
			request: []string{
				"bob",
				"create",
				"instances",
				"dev/*/*",
			},
			can: true,
		},
		{
			request: []string{
				"bob",
				"create",
				"instances",
				"dev/default/bob-1",
			},
			can: true,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			t.Parallel()
			can, err := Can(context.Background(), "./testdata/policy-1-good.csv", nil, tc.request...)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if can != tc.can {
				t.Fatalf("expected %v, got %v", tc.can, can)
			}
		})
	}
}

func TestRBACName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ns/obj", ObjectName("ns", "obj"))
	assert.Equal(t, "ns/", ObjectName("ns", ""))
	assert.Equal(t, "/", ObjectName("", ""))
	assert.Equal(t, "ns", ObjectName("ns"))
}
