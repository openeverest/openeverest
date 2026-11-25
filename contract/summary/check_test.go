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

package summary

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	yes     = metav1.ConditionTrue
	no      = metav1.ConditionFalse
	unknown = metav1.ConditionUnknown
)

func TestCheckConditions(t *testing.T) {
	t.Parallel()

	cond := func(condType string, status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Type: condType, Status: status, Reason: reason}
	}
	// with replaces the conditions of the same types.
	with := func(base []metav1.Condition, changes ...metav1.Condition) []metav1.Condition {
		out := slices.Clone(base)
		for _, c := range changes {
			i := slices.IndexFunc(out, func(have metav1.Condition) bool { return have.Type == c.Type })
			if i < 0 {
				out = append(out, c)

				continue
			}
			out[i] = c
		}

		return out
	}
	ready := []metav1.Condition{
		cond(v1alpha1.ConditionAvailable, yes, v1alpha1.ReasonAvailable),
		cond(v1alpha1.ConditionReady, yes, v1alpha1.ReasonReady),
		cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonConverged),
		cond(v1alpha1.ConditionStalled, no, v1alpha1.ReasonConverged),
		cond(v1alpha1.ConditionNeedsAttention, no, v1alpha1.ReasonNoProblems),
	}
	creating := []metav1.Condition{
		cond(v1alpha1.ConditionAvailable, no, v1alpha1.ReasonNotCreated),
		cond(v1alpha1.ConditionReady, no, v1alpha1.ReasonCreating),
		cond(v1alpha1.ConditionReconciling, yes, v1alpha1.ReasonCreating),
		cond(v1alpha1.ConditionStalled, no, v1alpha1.ReasonReconciling),
		cond(v1alpha1.ConditionNeedsAttention, no, v1alpha1.ReasonNoProblems),
	}
	crashLoopStalled := []metav1.Condition{
		cond(v1alpha1.ConditionAvailable, no, v1alpha1.ReasonNotAvailable),
		cond(v1alpha1.ConditionReady, no, v1alpha1.ReasonCrashLoopBackOff),
		cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonStalled),
		cond(v1alpha1.ConditionStalled, yes, v1alpha1.ReasonCrashLoopBackOff),
		cond(v1alpha1.ConditionNeedsAttention, yes, v1alpha1.ReasonCrashLoopBackOff),
	}

	tests := []struct {
		name  string
		conds []metav1.Condition
		// wantErrs are parts of the error, one per rule broken.
		wantErrs []string
	}{
		{
			name: "never reconciled",
		},
		{
			name:  "never reconciled, with a condition core writes",
			conds: []metav1.Condition{cond(v1alpha1.ConditionUnattended, unknown, v1alpha1.ReasonNotResponding)},
		},
		{
			name:  "ready",
			conds: ready,
		},
		{
			name:  "ready, while an operation the engine started supplies the word",
			conds: with(ready, cond(v1alpha1.ConditionReady, yes, v1alpha1.ConditionStorageResizing)),
		},
		{
			name:  "working",
			conds: creating,
		},
		{
			name: "working on a problem",
			conds: with(creating,
				cond(v1alpha1.ConditionReady, no, v1alpha1.ReasonUserSecretNotFound),
				cond(v1alpha1.ConditionReconciling, yes, v1alpha1.ReasonUserSecretNotFound),
				cond(v1alpha1.ConditionNeedsAttention, yes, v1alpha1.ReasonUserSecretNotFound),
			),
		},
		{
			name:  "stalled",
			conds: crashLoopStalled,
		},
		{
			name: "stalled by the progress deadline, under the problem's name",
			conds: with(crashLoopStalled,
				cond(v1alpha1.ConditionStalled, yes, v1alpha1.ReasonProgressDeadlineExceeded),
			),
		},
		{
			name: "some summary conditions are missing",
			conds: []metav1.Condition{
				cond(v1alpha1.ConditionReady, yes, v1alpha1.ReasonReady),
				cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonConverged),
			},
			wantErrs: []string{"summary conditions [Stalled NeedsAttention] are missing"},
		},
		{
			name: "the reasons written before False ones said why",
			conds: with(ready,
				cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonNoProblems),
				cond(v1alpha1.ConditionStalled, no, v1alpha1.ReasonNoProblems),
			),
			wantErrs: []string{
				`Reconciling=False has reason "NoProblems", want one of [Converged Stalled]`,
				`Stalled=False has reason "NoProblems", want one of [Converged Reconciling]`,
				`Reconciling=False has reason "NoProblems" while Ready=True, want Converged exactly while Ready=True`,
				`Stalled=False has reason "NoProblems" while Ready=True, want Converged exactly while Ready=True`,
				"Reconciling=False has reason NoProblems, which only NeedsAttention=False carries",
				"Stalled=False has reason NoProblems, which only NeedsAttention=False carries",
			},
		},
		{
			name: "a stall named after its cause on Reconciling",
			conds: with(crashLoopStalled,
				cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonCrashLoopBackOff),
			),
			wantErrs: []string{
				`Reconciling=False has reason "CrashLoopBackOff", want one of [Converged Stalled]`,
				`Reconciling=False has reason "CrashLoopBackOff" while Stalled=True, want Reconciling=False Stalled exactly while Stalled=True`,
			},
		},
		{
			name: "Converged while not Ready",
			conds: with(creating,
				cond(v1alpha1.ConditionStalled, no, v1alpha1.ReasonConverged),
			),
			wantErrs: []string{`Stalled=False has reason "Converged" while Ready=False, want Converged exactly while Ready=True`},
		},
		{
			name: "Stalled on Reconciling while not stalled",
			conds: with(ready,
				cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonStalled),
			),
			wantErrs: []string{
				`Reconciling=False has reason "Stalled" while Ready=True, want Converged exactly while Ready=True`,
				`Reconciling=False has reason "Stalled" while Stalled=False, want Reconciling=False Stalled exactly while Stalled=True`,
			},
		},
		{
			name: "NeedsAttention False with another reason",
			conds: with(ready,
				cond(v1alpha1.ConditionNeedsAttention, no, v1alpha1.ReasonHealthy),
			),
			wantErrs: []string{`NeedsAttention=False has reason "Healthy", want one of [NoProblems]`},
		},
		{
			name: "NoProblems on Available",
			conds: with(ready,
				cond(v1alpha1.ConditionAvailable, yes, v1alpha1.ReasonNoProblems),
			),
			wantErrs: []string{"Available=True has reason NoProblems, which only NeedsAttention=False carries"},
		},
		{
			name: "both True",
			conds: with(crashLoopStalled,
				cond(v1alpha1.ConditionReconciling, yes, v1alpha1.ReasonCrashLoopBackOff),
			),
			wantErrs: []string{
				"Ready=False with Reconciling=True and Stalled=True, want exactly one of them True",
				`Reconciling=True has reason "CrashLoopBackOff" while Stalled=True, want Reconciling=False Stalled exactly while Stalled=True`,
			},
		},
		{
			name: "neither True while not Ready",
			conds: with(creating,
				cond(v1alpha1.ConditionReconciling, no, v1alpha1.ReasonStalled),
			),
			wantErrs: []string{
				"Ready=False with Reconciling=False and Stalled=False, want exactly one of them True",
				`Reconciling=False has reason "Stalled" while Stalled=False`,
			},
		},
		{
			name: "a problem while Ready",
			conds: with(ready,
				cond(v1alpha1.ConditionNeedsAttention, yes, v1alpha1.ReasonReady),
			),
			wantErrs: []string{"Ready=True with Reconciling=False, Stalled=False and NeedsAttention=True, want all three False"},
		},
		{
			name: "Unknown",
			conds: with(creating,
				cond(v1alpha1.ConditionNeedsAttention, unknown, v1alpha1.ReasonNoProblems),
			),
			wantErrs: []string{
				"NeedsAttention is Unknown, want True or False",
				"NeedsAttention=Unknown has reason NoProblems, which only NeedsAttention=False carries",
			},
		},
		{
			name: "True with another cause than Ready's",
			conds: with(crashLoopStalled,
				cond(v1alpha1.ConditionStalled, yes, v1alpha1.ReasonUnschedulable),
				cond(v1alpha1.ConditionNeedsAttention, yes, v1alpha1.ReasonUnschedulable),
			),
			wantErrs: []string{
				`NeedsAttention=True has reason "Unschedulable", want Ready's "CrashLoopBackOff"`,
				`Stalled=True has reason "Unschedulable", want Ready's "CrashLoopBackOff" or ProgressDeadlineExceeded`,
			},
		},
		{
			name: "Reconciling with another cause than Ready's",
			conds: with(creating,
				cond(v1alpha1.ConditionReconciling, yes, v1alpha1.ReasonApplyingChanges),
			),
			wantErrs: []string{`Reconciling=True has reason "ApplyingChanges", want Ready's "Creating"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckConditions(tt.conds)

			if len(tt.wantErrs) == 0 {
				require.NoError(t, err)

				return
			}
			require.Error(t, err)
			for _, want := range tt.wantErrs {
				assert.Contains(t, err.Error(), want)
			}
			errs := []error{err}
			var joined interface{ Unwrap() []error }
			if errors.As(err, &joined) {
				errs = joined.Unwrap()
			}
			assert.Len(t, errs, len(tt.wantErrs), "one error per rule broken: %v", err)
		})
	}
}
