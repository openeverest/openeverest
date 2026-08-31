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

package v1alpha1_test

import (
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsinternal "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	schemacel "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	apivalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Mirrors of the apiserver CEL cost constants (k8s.io/apiserver/pkg/apis/cel).
// They are inlined so that these tests do not promote k8s.io/apiserver from an
// indirect to a direct module dependency.
const (
	celPerCallLimit         uint64 = 1000000
	celRuntimeCELCostBudget int64  = 10000000
)

// crdPath is the generated Instance CRD. Validating against the generated
// artifact (rather than a hand-written copy of the rules) keeps these tests
// honest: if the kubebuilder markers on InstanceBackupScheduleRetention
// change without regenerating, or regenerate into something different than
// intended, these tests fail.
const crdPath = "../../../config/crd/bases/core.openeverest.io_instances.yaml"

// loadInstanceSchema reads the generated Instance CRD and returns the
// v1alpha1 root schema.
func loadInstanceSchema(t *testing.T) *apiextensionsv1.JSONSchemaProps {
	t.Helper()

	raw, err := os.ReadFile(crdPath)
	require.NoError(t, err, "failed to read the generated Instance CRD")

	var crd apiextensionsv1.CustomResourceDefinition
	require.NoError(t, yaml.Unmarshal(raw, &crd))

	for i := range crd.Spec.Versions {
		if crd.Spec.Versions[i].Name == "v1alpha1" && crd.Spec.Versions[i].Schema != nil {
			return crd.Spec.Versions[i].Schema.OpenAPIV3Schema
		}
	}

	require.FailNow(t, "v1alpha1 schema not found in the generated Instance CRD")

	return nil
}

// loadScheduleSchema returns the schema of a single
// spec.backup.storages[].schedules[] entry.
func loadScheduleSchema(t *testing.T) *apiextensionsv1.JSONSchemaProps {
	t.Helper()

	root := loadInstanceSchema(t)

	spec, ok := root.Properties["spec"]
	require.True(t, ok, "spec not found")
	backup, ok := spec.Properties["backup"]
	require.True(t, ok, "spec.backup not found")
	storages, ok := backup.Properties["storages"]
	require.True(t, ok, "spec.backup.storages not found")
	require.NotNil(t, storages.Items)
	require.NotNil(t, storages.Items.Schema)
	schedules, ok := storages.Items.Schema.Properties["schedules"]
	require.True(t, ok, "spec.backup.storages[].schedules not found")
	require.NotNil(t, schedules.Items)
	require.NotNil(t, schedules.Items.Schema)

	return schedules.Items.Schema
}

// loadRetentionSchema extracts the schema for a single
// spec.backup.storages[].schedules[].retention object out of the generated
// Instance CRD, and builds both an OpenAPI validator (for enum, minimum and
// pattern) and a structural schema (for the x-kubernetes-validations CEL
// rules).
//
//nolint:ireturn // apivalidation.NewSchemaValidator only returns an interface.
func loadRetentionSchema(t *testing.T) (*structuralschema.Structural, apivalidation.SchemaValidator) {
	t.Helper()

	schedule := loadScheduleSchema(t)

	retention, ok := schedule.Properties["retention"]
	require.True(t, ok, "spec.backup.storages[].schedules[].retention not found")

	internal := &apiextensionsinternal.JSONSchemaProps{}
	require.NoError(t, apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
		&retention, internal, nil,
	))

	structural, err := structuralschema.NewStructural(internal)
	require.NoError(t, err)

	validator, _, err := apivalidation.NewSchemaValidator(internal)
	require.NoError(t, err)

	return structural, validator
}

// validateRetention runs both OpenAPI structural validation and the CEL
// x-kubernetes-validations rules against a retention object, returning all
// error messages.
func validateRetention(t *testing.T, obj map[string]any) []string {
	t.Helper()

	structural, validator := loadRetentionSchema(t)

	var msgs []string
	if res := validator.Validate(obj); res != nil && len(res.Errors) > 0 {
		for _, err := range res.Errors {
			msgs = append(msgs, err.Error())
		}
	}

	celValidator := schemacel.NewValidator(structural, true, celPerCallLimit)
	if celValidator != nil {
		errs, _ := celValidator.Validate(
			t.Context(),
			field.NewPath("retention"),
			structural,
			obj,
			nil,
			celRuntimeCELCostBudget,
		)
		for _, err := range errs {
			msgs = append(msgs, err.Error())
		}
	}

	return msgs
}

func TestInstanceBackupScheduleRetentionCRDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// retention is the object as it would appear in the Instance YAML.
		retention map[string]any
		// wantErrContains, when non-empty, requires validation to fail with
		// a message containing this substring.
		wantErrContains string
	}{
		{
			name:      "valid count retention",
			retention: map[string]any{"type": "count", "count": int64(3)},
		},
		{
			name:      "valid time retention in days",
			retention: map[string]any{"type": "time", "duration": "7d"},
		},
		{
			name:      "valid time retention in weeks",
			retention: map[string]any{"type": "time", "duration": "2w"},
		},
		{
			name:      "valid time retention in months",
			retention: map[string]any{"type": "time", "duration": "6m"},
		},
		{
			name:      "multi digit duration is valid",
			retention: map[string]any{"type": "time", "duration": "30d"},
		},
		{
			name:            "count type rejects duration",
			retention:       map[string]any{"type": "count", "count": int64(3), "duration": "7d"},
			wantErrContains: "retention type 'count' requires .count and forbids .duration",
		},
		{
			name:            "count type requires count",
			retention:       map[string]any{"type": "count"},
			wantErrContains: "retention type 'count' requires .count and forbids .duration",
		},
		{
			name:            "time type rejects count",
			retention:       map[string]any{"type": "time", "duration": "7d", "count": int64(3)},
			wantErrContains: "retention type 'time' requires .duration and forbids .count",
		},
		{
			name:            "time type requires duration",
			retention:       map[string]any{"type": "time"},
			wantErrContains: "retention type 'time' requires .duration and forbids .count",
		},
		{
			name:            "unknown type is rejected",
			retention:       map[string]any{"type": "forever", "count": int64(3)},
			wantErrContains: "should be one of [count time]",
		},
		{
			name:            "zero count is rejected",
			retention:       map[string]any{"type": "count", "count": int64(0)},
			wantErrContains: "count",
		},
		{
			name:            "negative count is rejected",
			retention:       map[string]any{"type": "count", "count": int64(-1)},
			wantErrContains: "count",
		},
		{
			name:            "duration without unit is rejected",
			retention:       map[string]any{"type": "time", "duration": "7"},
			wantErrContains: "duration",
		},
		{
			name:            "duration with unsupported unit is rejected",
			retention:       map[string]any{"type": "time", "duration": "7y"},
			wantErrContains: "duration",
		},
		{
			name:            "duration with hours is rejected",
			retention:       map[string]any{"type": "time", "duration": "24h"},
			wantErrContains: "duration",
		},
		{
			name:            "duration with unit only is rejected",
			retention:       map[string]any{"type": "time", "duration": "d"},
			wantErrContains: "duration",
		},
		{
			name:            "empty duration is rejected",
			retention:       map[string]any{"type": "time", "duration": ""},
			wantErrContains: "duration",
		},
		// A zero-length retention window would mean "retain nothing", i.e.
		// delete every backup as soon as it is taken. The pattern requires a
		// non-zero leading digit so these cannot be expressed at all.
		{
			name:            "zero day duration is rejected",
			retention:       map[string]any{"type": "time", "duration": "0d"},
			wantErrContains: "duration",
		},
		{
			name:            "zero week duration with leading zeros is rejected",
			retention:       map[string]any{"type": "time", "duration": "00w"},
			wantErrContains: "duration",
		},
		{
			name:            "zero month duration is rejected",
			retention:       map[string]any{"type": "time", "duration": "0m"},
			wantErrContains: "duration",
		},
		{
			name:            "leading zero duration is rejected",
			retention:       map[string]any{"type": "time", "duration": "07d"},
			wantErrContains: "duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msgs := validateRetention(t, tt.retention)

			if tt.wantErrContains == "" {
				assert.Empty(t, msgs, "expected %v to be valid", tt.retention)
				return
			}

			require.NotEmpty(t, msgs, "expected %v to be rejected", tt.retention)
			assert.Contains(t, strings.Join(msgs, "; "), tt.wantErrContains,
				"expected an error containing %q, got: %v", tt.wantErrContains, msgs,
			)
		})
	}
}

// TestInstanceBackupScheduleRetentionDefaulting documents that the CRD
// defaults .type to "count", so a retention block that only sets .count is
// accepted once defaulting has run.
func TestInstanceBackupScheduleRetentionDefaulting(t *testing.T) {
	t.Parallel()

	structural, _ := loadRetentionSchema(t)

	typeProp, ok := structural.Properties["type"]
	require.True(t, ok, "retention.type must exist in the generated CRD")
	require.NotNil(t, typeProp.Default.Object, "retention.type must carry a default")
	assert.Equal(t, "count", typeProp.Default.Object)
}

// TestInstanceBackupScheduleRetentionDefaultingThenValidation runs the real
// apiserver ordering — structural defaulting first, then schema + CEL
// validation — against retention objects that omit .type.
//
// This is the case that matters most: because .type defaults to "count", a
// user who writes only a .duration does NOT get time-based retention. They
// get type=count with no .count, which the CEL rule must reject. If that rule
// ever stops firing, the object would be admitted and resolve to a count
// policy with Count 0 — "delete everything". These cases pin that shut.
func TestInstanceBackupScheduleRetentionDefaultingThenValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// retention is written by the user, before defaulting.
		retention map[string]any
		// wantType is the value of .type after defaulting has run.
		wantType string
		// wantErrContains, when non-empty, requires post-defaulting
		// validation to fail with a message containing this substring.
		wantErrContains string
	}{
		{
			// The happy path the default exists for: count is implied.
			name:      "count only defaults type to count and is valid",
			retention: map[string]any{"count": int64(3)},
			wantType:  "count",
		},
		{
			// The escape hatch. Defaulting makes this type=count, and
			// count is absent, so the count CEL rule must reject it.
			// The user has to say type: time explicitly.
			name:            "duration only defaults to count and is then rejected",
			retention:       map[string]any{"duration": "7d"},
			wantType:        "count",
			wantErrContains: "retention type 'count' requires .count and forbids .duration",
		},
		{
			// Empty object: defaults to count with no count → rejected,
			// rather than silently becoming "keep 0".
			name:            "empty retention object defaults to count and is then rejected",
			retention:       map[string]any{},
			wantType:        "count",
			wantErrContains: "retention type 'count' requires .count and forbids .duration",
		},
		{
			// Both fields, no type: defaulting picks count, and the count
			// rule forbids .duration.
			name:            "count and duration without type defaults to count and is then rejected",
			retention:       map[string]any{"count": int64(3), "duration": "7d"},
			wantType:        "count",
			wantErrContains: "retention type 'count' requires .count and forbids .duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			structural, _ := loadRetentionSchema(t)

			// Copy so the table entry is not mutated by defaulting.
			obj := make(map[string]any, len(tt.retention))
			maps.Copy(obj, tt.retention)

			structuraldefaulting.Default(obj, structural)

			assert.Equal(t, tt.wantType, obj["type"],
				"defaulting should have set .type")

			msgs := validateRetention(t, obj)

			if tt.wantErrContains == "" {
				assert.Empty(t, msgs, "expected %v to be valid after defaulting", obj)
				return
			}

			require.NotEmpty(t, msgs,
				"expected %v to be rejected after defaulting — an accepted "+
					"object here would resolve to a count policy with count 0", obj)
			assert.Contains(t, strings.Join(msgs, "; "), tt.wantErrContains,
				"expected an error containing %q, got: %v", tt.wantErrContains, msgs,
			)
		})
	}
}

// TestRetentionCopiesStillPresent guards the backward-compatibility promise:
// this change must not remove or alter the legacy retentionCopies field.
func TestRetentionCopiesStillPresent(t *testing.T) {
	t.Parallel()

	schedule := loadScheduleSchema(t)

	retentionCopies, ok := schedule.Properties["retentionCopies"]
	require.True(t, ok, "retentionCopies must not be removed")
	assert.Equal(t, "integer", retentionCopies.Type)
	require.NotNil(t, retentionCopies.Minimum, "retentionCopies must keep its minimum")
	assert.InDelta(t, 0.0, *retentionCopies.Minimum, 1e-9,
		"retentionCopies must keep minimum 0 so existing objects stay valid")

	// retentionCopies must remain optional.
	assert.NotContains(t, schedule.Required, "retentionCopies")
	// retention must be optional too, so existing schedules keep validating.
	assert.NotContains(t, schedule.Required, "retention")
}
