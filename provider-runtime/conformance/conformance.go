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

// Package conformance exercises a provider against the contract the runtime
// expects of it, without a cluster.
//
// It is meant to be called from a provider's own test suite:
//
//	func TestUISchemaIsReconciled(t *testing.T) {
//	    conformance.UISchemaIsReconciled(t, conformance.Config{
//	        Provider: NewMyProvider(),
//	    })
//	}
package conformance

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// Config configures a conformance run.
type Config struct {
	// Provider is the implementation under test.
	Provider ProviderUnderTest

	// ProviderSpecPath overrides the generated Provider CR to check against.
	// When empty it is located by walking up to the module root and looking
	// for charts/*/generated/provider-spec.yaml.
	ProviderSpecPath string

	// Unverifiable maps a UI schema path to the reason the harness cannot
	// observe it. Use it when a provider reshapes a value so thoroughly that
	// it is unrecognisable in the resulting engine CR — not to silence a
	// field that is genuinely unhandled.
	Unverifiable map[string]string
}

// outcome is what the harness observed for one UI schema path.
type outcome int

const (
	// reconciled means the value reached an object the provider applied.
	reconciled outcome = iota
	// read means the value appeared in a request the provider made, which
	// proves the field was consumed even though it never reaches the engine CR.
	read
	// ignored means nothing observable happened.
	ignored
	// unverified means the render failed, so nothing can be concluded.
	unverified
)

type result struct {
	topology string
	path     string
	outcome  outcome
	detail   string
	requests []string
}

// UISchemaIsReconciled asserts that every field the UI schema offers is
// actually consumed by the provider. A form field bound to a path the provider
// ignores accepts input and silently does nothing, which is worse than not
// offering it at all.
func UISchemaIsReconciled(t *testing.T, cfg Config) {
	t.Helper()

	spec, err := loadProviderSpec(cfg.ProviderSpecPath)
	if err != nil {
		t.Fatalf("conformance: %v", err)
	}

	var results []result
	for _, topology := range sortedKeys(spec.Topologies) {
		paths, err := uiPaths(spec, topology)
		if err != nil {
			t.Fatalf("conformance: reading UI schema for topology %q: %v", topology, err)
		}
		if len(paths) == 0 {
			continue
		}
		results = append(results, probeTopology(t, cfg, spec, topology, paths)...)
	}

	report(t, cfg, results, uiSchemaSource())
}

// source describes where a probed path came from, so a failure names the
// declaration the author has to fix.
type source struct {
	subject string
	ignored string
}

func uiSchemaSource() source {
	return source{
		subject: "UI schema references",
		ignored: "offered by the UI, but setting it changes nothing the provider applies or requests",
	}
}

func report(t *testing.T, cfg Config, results []result, src source) {
	t.Helper()

	var failures, notes []string
	for _, r := range results {
		if reason, ok := cfg.Unverifiable[r.path]; ok {
			notes = append(notes, fmt.Sprintf("  %s %s\n    declared unverifiable: %s", r.topology, r.path, reason))
			continue
		}
		switch r.outcome {
		case reconciled, read:
			continue
		case ignored:
			failures = append(failures, fmt.Sprintf(
				"  topology %s\n    %s\n      %s",
				r.topology, r.path, src.ignored))
		case unverified:
			failures = append(failures, fmt.Sprintf(
				"  topology %s\n    %s\n      could not be verified: %s%s",
				r.topology, r.path, r.detail, formatRequests(r.requests)))
		}
	}

	for _, note := range notes {
		t.Log(note)
	}
	if len(failures) > 0 {
		t.Errorf("%s %d field(s) this provider does not reconcile:\n%s",
			src.subject, len(failures), strings.Join(failures, "\n"))
	}
}

func formatRequests(requests []string) string {
	if len(requests) == 0 {
		return ""
	}
	return "\n      requests observed: " + strings.Join(requests, ", ")
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
