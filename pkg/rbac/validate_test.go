// everest
// Copyright (C) 2023 Percona LLC
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
				{"role:admin", "database-clusters", "create", "*"},
				{"role:admin", "monitoring-instances", "*", "*"},
			},
			valid: true,
		},
		{
			policies: [][]string{
				{"role:admin", "database-clusters", "create", "*"},
				{"role:admin", "monitoring-instances", "*", "*"},
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
				{"role:admin", "database-clusters", "create", "*"},
				{"role:admin", "monitoring-instances", "*", "*"},
			},
			valid: true,
		},
		{
			roles: []string{"role:admin", "role:viewer"},
			policies: [][]string{
				{"role:admin", "database-clusters", "create", "*"},
				{"role:admin", "monitoring-instances", "*", "*"},
				{"role:does-not-exist", "monitoring-instances", "*", "*"},
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
			terms: []string{"role:admin", "database-clusters", "create", "*"},
			valid: true,
		},
		{
			terms: []string{"alice@corp.io", "database-clusters", "read", "ex-1/mycompany.com"},
			valid: true,
		},
		{
			terms: []string{"auth0|1a2b3c", "database-clusters", "update", "*/*"},
			valid: true,
		},
		{
			terms: []string{"role:team-dev", "database-clusters", "delete", "res/app-[0-9]"},
			valid: true,
		},
		{
			terms: []string{"role:team-dev", "database-clusters", "read", "ns/*"},
			valid: true,
		},
		{
			terms: []string{"role:", "database-clusters", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin user", "database-clusters", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"", "database-clusters", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin", "", "create", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin", "database-clusters", "invalid_action", "*"},
			valid: false,
		},
		{
			terms: []string{"role:admin", "database-clusters", "create", "ex/my-[cluster"},
			valid: false,
		},
		{
			terms: []string{"role:admin", "database-clusters", "create", "a/b/c"},
			valid: false,
		},
		{
			terms: []string{"role:admin", "database-clusters", "create"},
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
				"database-clusters",
				"*",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"read",
				"database-clusters",
				"*",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"update",
				"database-clusters",
				"*",
			},
			can: true,
		},
		{
			request: []string{
				"admin",
				"update",
				"database-cluster-backups",
				"*",
			},
			can: true,
		},
		{
			request: []string{
				"alice",
				"create",
				"database-clusters",
				"*",
			},
			can: false,
		},
		{
			request: []string{
				"alice",
				"read",
				"database-engines",
				"*",
			},
			can: true,
		},
		{
			request: []string{
				"alice",
				"create",
				"database-clusters",
				"alice/alice-cluster-1",
			},
			can: true,
		},
		{
			request: []string{
				"bob",
				"create",
				"database-clusters",
				"*",
			},
			can: false,
		},
		{
			request: []string{
				"bob",
				"create",
				"database-clusters",
				"dev/*",
			},
			can: true,
		},
		{
			request: []string{
				"bob",
				"create",
				"database-clusters",
				"dev/bob-1",
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
