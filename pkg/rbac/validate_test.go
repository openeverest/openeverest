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
	"github.com/stretchr/testify/require"
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
		name  string
		terms []string
		valid bool
	}{
		{
			name:  "basic policy row",
			terms: []string{"role:admin", "instances", "create", "*"},
			valid: true,
		},
		{
			name:  "email subject stays valid",
			terms: []string{"alice@example.com", "instances", "read", "prod/dev/db"},
			valid: true,
		},
		{
			name:  "dotted email subject",
			terms: []string{"jane.doe@corp.io", "backups", "read", "*/dev/*"},
			valid: true,
		},
		{
			// `+` subaddressing was accepted pre-fix (via the range) and is common
			name:  "plus-addressed email subject",
			terms: []string{"alice+everest@example.com", "instances", "read", "*"},
			valid: true,
		},
		{
			name:  "bang in subject",
			terms: []string{"role:admin!!", "instances", "create", "*"},
			valid: false,
		},
		{
			name:  "space in resource",
			terms: []string{"role:admin", "instances names", "create", "*"},
			valid: false,
		},
		{
			name:  "empty term",
			terms: []string{"role:admin", "", "create", "*"},
			valid: false,
		},
		{
			// `[` was admitted by the old `*-_` range and produces
			// ErrBadPattern at enforce time — every request 500s.
			name:  "glob class in object",
			terms: []string{"role:admin", "instances", "read", "prod/[db"},
			valid: false,
		},
		{
			name:  "range-admitted metacharacters in object",
			terms: []string{"role:admin", "instances", "read", "prod/db?;=<>@+,\\]^"},
			valid: false,
		},
		{
			name:  "at-sign in resource",
			terms: []string{"role:admin", "instances@", "read", "*"},
			valid: false,
		},
		{
			name:  "uppercase in resource",
			terms: []string{"role:admin", "Instances", "read", "*"},
			valid: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateTerms(tc.terms)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
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
