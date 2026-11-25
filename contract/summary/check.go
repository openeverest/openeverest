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

// Package summary holds the rules the summary conditions of an Instance
// keep: Ready, Reconciling, Stalled and NeedsAttention. The provider runtime
// writes them by these rules, and every reader of an Instance's status, such
// as the UI, everestctl or a GitOps tool, relies on them to tell a converged
// Instance from one that is being worked on or needs help.
package summary

import (
	"errors"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// CheckConditions returns every way the summary conditions break the rules
// Compute writes them by, or nil when they keep them all. An Instance no
// provider has reconciled has no summary conditions and passes.
//
// The rules are a closed list on Reconciling, Stalled and NeedsAttention.
// Ready and Available carry causes, which are open-ended.
//   - While Ready is True, Reconciling and Stalled are False with reason
//     Converged, and NeedsAttention is False.
//   - While Ready is not True, exactly one of Reconciling and Stalled is
//     True. Reconciling then carries Ready's reason, and Stalled is False
//     with reason Reconciling. Stalled carries Ready's reason or
//     ProgressDeadlineExceeded, and Reconciling is False with reason Stalled.
//   - NeedsAttention is True with Ready's reason, or False with NoProblems,
//     which no other summary condition carries.
func CheckConditions(conds []metav1.Condition) error {
	s := summaryConditions{
		available:   meta.FindStatusCondition(conds, v1alpha1.ConditionAvailable),
		ready:       meta.FindStatusCondition(conds, v1alpha1.ConditionReady),
		reconciling: meta.FindStatusCondition(conds, v1alpha1.ConditionReconciling),
		stalled:     meta.FindStatusCondition(conds, v1alpha1.ConditionStalled),
		attention:   meta.FindStatusCondition(conds, v1alpha1.ConditionNeedsAttention),
	}
	missing := s.missing()
	switch len(missing) {
	case 0:
	case 4:
		// Never reconciled.
		return nil
	default:
		return fmt.Errorf("summary conditions %v are missing", missing)
	}

	return errors.Join(slices.Concat(
		s.checkStatuses(), s.checkFalseReasons(), s.checkTrueReasons(), s.checkNoProblems(),
	)...)
}

// summaryConditions are the conditions the rules are about.
type summaryConditions struct {
	available, ready, reconciling, stalled, attention *metav1.Condition
}

// missing returns the types of the conditions the rules need that are
// absent. Available is not among them.
func (s *summaryConditions) missing() []string {
	var missing []string
	for _, c := range []struct {
		condType  string
		condition *metav1.Condition
	}{
		{v1alpha1.ConditionReady, s.ready},
		{v1alpha1.ConditionReconciling, s.reconciling},
		{v1alpha1.ConditionStalled, s.stalled},
		{v1alpha1.ConditionNeedsAttention, s.attention},
	} {
		if c.condition == nil {
			missing = append(missing, c.condType)
		}
	}

	return missing
}

// checkStatuses checks that a tool reading Reconciling and Stalled finds
// exactly one True while the Instance is not Ready, and none while it is.
func (s *summaryConditions) checkStatuses() []error {
	var errs []error
	for _, c := range []*metav1.Condition{s.reconciling, s.stalled, s.attention} {
		if c.Status != metav1.ConditionTrue && c.Status != metav1.ConditionFalse {
			errs = append(errs, fmt.Errorf("%s is %s, want True or False", c.Type, c.Status))
		}
	}
	reconciling, stalled := isTrue(s.reconciling), isTrue(s.stalled)
	switch {
	case isTrue(s.ready) && (reconciling || stalled || isTrue(s.attention)):
		errs = append(errs, fmt.Errorf("%s=True with Reconciling=%s, Stalled=%s and NeedsAttention=%s, want all three False",
			s.ready.Type, s.reconciling.Status, s.stalled.Status, s.attention.Status,
		))
	case !isTrue(s.ready) && reconciling == stalled:
		errs = append(errs, fmt.Errorf("%s=%s with Reconciling=%s and Stalled=%s, want exactly one of them True",
			s.ready.Type, s.ready.Status, s.reconciling.Status, s.stalled.Status,
		))
	}

	return errs
}

// checkFalseReasons checks that a False condition says why it is False, and
// that the reason agrees with the other conditions.
func (s *summaryConditions) checkFalseReasons() []error {
	var errs []error
	for _, c := range []struct {
		condition *metav1.Condition
		reasons   []string
	}{
		{s.reconciling, []string{v1alpha1.ReasonConverged, v1alpha1.ReasonStalled}},
		{s.stalled, []string{v1alpha1.ReasonConverged, v1alpha1.ReasonReconciling}},
		{s.attention, []string{v1alpha1.ReasonNoProblems}},
	} {
		if c.condition.Status == metav1.ConditionFalse && !slices.Contains(c.reasons, c.condition.Reason) {
			errs = append(errs, fmt.Errorf("%s=False has reason %q, want one of %v",
				c.condition.Type, c.condition.Reason, c.reasons,
			))
		}
	}
	for _, c := range []*metav1.Condition{s.reconciling, s.stalled} {
		if (c.Reason == v1alpha1.ReasonConverged) != isTrue(s.ready) {
			errs = append(errs, fmt.Errorf("%s=%s has reason %q while Ready=%s, want %s exactly while Ready=True",
				c.Type, c.Status, c.Reason, s.ready.Status, v1alpha1.ReasonConverged,
			))
		}
	}
	stalledReason := s.reconciling.Status == metav1.ConditionFalse && s.reconciling.Reason == v1alpha1.ReasonStalled
	if stalledReason != isTrue(s.stalled) {
		errs = append(errs, fmt.Errorf("%s=%s has reason %q while Stalled=%s, want Reconciling=False %s exactly while Stalled=True",
			s.reconciling.Type, s.reconciling.Status, s.reconciling.Reason, s.stalled.Status, v1alpha1.ReasonStalled,
		))
	}

	return errs
}

// checkTrueReasons checks that a True condition names the cause Ready does.
func (s *summaryConditions) checkTrueReasons() []error {
	var errs []error
	for _, c := range []*metav1.Condition{s.reconciling, s.attention} {
		if isTrue(c) && c.Reason != s.ready.Reason {
			errs = append(errs, fmt.Errorf("%s=True has reason %q, want Ready's %q", c.Type, c.Reason, s.ready.Reason))
		}
	}
	if isTrue(s.stalled) && s.stalled.Reason != s.ready.Reason && s.stalled.Reason != v1alpha1.ReasonProgressDeadlineExceeded {
		errs = append(errs, fmt.Errorf("%s=True has reason %q, want Ready's %q or %s",
			s.stalled.Type, s.stalled.Reason, s.ready.Reason, v1alpha1.ReasonProgressDeadlineExceeded,
		))
	}

	return errs
}

// checkNoProblems checks that NoProblems is only on NeedsAttention=False.
func (s *summaryConditions) checkNoProblems() []error {
	var errs []error
	for _, c := range []*metav1.Condition{s.available, s.ready, s.reconciling, s.stalled, s.attention} {
		if c == nil || c.Reason != v1alpha1.ReasonNoProblems {
			continue
		}
		if c != s.attention || c.Status != metav1.ConditionFalse {
			errs = append(errs, fmt.Errorf("%s=%s has reason %s, which only NeedsAttention=False carries",
				c.Type, c.Status, v1alpha1.ReasonNoProblems,
			))
		}
	}

	return errs
}

func isTrue(c *metav1.Condition) bool {
	return c.Status == metav1.ConditionTrue
}
