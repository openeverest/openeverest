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

package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/instancestatus"
)

// StatusOptions holds the inputs for the status command.
type StatusOptions struct {
	Name      string
	Namespace string
	Cluster   string
	Context   string
	Watch     bool
	Interval  time.Duration
}

// InstanceStatusRunner fetches and prints the status of an instance.
type InstanceStatusRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewInstanceStatusRunner creates a new InstanceStatusRunner.
func NewInstanceStatusRunner(cfg Config, l *zap.SugaredLogger) *InstanceStatusRunner {
	is := &InstanceStatusRunner{config: cfg, l: l.With("component", "instance-status")}
	if cfg.Pretty {
		is.l = zap.NewNop().Sugar()
	}
	return is
}

// Run fetches the current status of an instance, printing it to stdout.
// With opts.Watch set, it polls continuously until the context is cancelled,
// the instance is deleted, or token refresh fails.
func (is *InstanceStatusRunner) Run(ctx context.Context, opts StatusOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: is.config.Pretty}, is.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch instance %q: %w", opts.Name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("instance %q not found in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return fmt.Errorf("unexpected response fetching instance %q: %s", opts.Name, resp.Status())
	}

	now := time.Now()
	var buf bytes.Buffer
	is.render(&buf, resp.JSON200, opts.Namespace, now)
	fmt.Fprint(os.Stdout, buf.String())

	if opts.Watch {
		return is.watch(ctx, c, opts, is.changeKey(resp.JSON200, opts.Namespace, now))
	}
	return nil
}

func (is *InstanceStatusRunner) watch(
	ctx context.Context,
	c *client.ClientWithResponses,
	opts StatusOptions,
	lastKey string,
) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	poll := newInstancePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			inst, err := poll(ctx)
			if err != nil {
				if _, ok := errors.AsType[*wait.RetryableError](err); ok {
					is.l.Warnf("%v — retrying in %s", err, interval)
					continue
				}
				return err
			}

			now := time.Now()
			key := is.changeKey(inst, opts.Namespace, now)
			if key == lastKey {
				continue
			}
			lastKey = key
			var buf bytes.Buffer
			is.render(&buf, inst, opts.Namespace, now)
			if is.config.Pretty {
				fmt.Fprintln(os.Stdout, "---")
			}
			fmt.Fprint(os.Stdout, buf.String())
		}
	}
}

// changeKey is what the watch compares between polls to decide whether to
// print again: what would be printed, leaving out how long things have
// lasted, which goes up on every poll.
func (is *InstanceStatusRunner) changeKey(inst *client.Instance, namespace string, now time.Time) string {
	var buf bytes.Buffer
	if !is.config.Pretty {
		is.render(&buf, inst, namespace, now)
		return buf.String()
	}
	writeInstanceStatus(&buf, inst, namespace, now, func(time.Time) string { return "" })
	return buf.String()
}

func (is *InstanceStatusRunner) render(w io.Writer, inst *client.Instance, namespace string, now time.Time) {
	if !is.config.Pretty {
		_ = json.NewEncoder(w).Encode(inst) //nolint:errchkjson
		return
	}
	printInstanceStatus(w, inst, namespace, now)
}

// printInstanceStatus prints what the instance's status says as of now.
func printInstanceStatus(w io.Writer, inst *client.Instance, namespace string, now time.Time) {
	writeInstanceStatus(w, inst, namespace, now, func(since time.Time) string { return age(since, now) })
}

// writeInstanceStatus prints what the instance's status says, most pressing
// first: the status in one line, then what is wrong, what is checked, what
// is running and what wants a decision, and last every condition as it is.
// ageOf says how long ago a time was.
func writeInstanceStatus(w io.Writer, inst *client.Instance, namespace string, now time.Time, ageOf func(time.Time) string) {
	in := statusInput(inst)
	shown := instancestatus.Describe(in, now)

	status := shown.Sentence
	if shown.Since != nil {
		status += fmt.Sprintf(" (for %s)", ageOf(*shown.Since))
	}
	message := "-"
	if shown.Message != "" {
		message = shown.Message
	}

	fmt.Fprintf(w, "Instance:  %s\n", instanceName(inst))
	fmt.Fprintf(w, "Namespace: %s\n", namespace)
	fmt.Fprintf(w, "Status:    %s\n", status)
	fmt.Fprintf(w, "Version:   %s\n", instanceVersionValue(inst))
	fmt.Fprintf(w, "Message:   %s\n", message)
	if note := staleNote(in, shown); note != "" {
		fmt.Fprintf(w, "Note:      %s\n", note)
	}

	printComponentTable(w, inst)
	printProblemTable(w, in.Conditions, ageOf)
	printCheckTable(w, in.Conditions)
	printRunningTable(w, "In progress", "OPERATION", kindOperation, in.Conditions, ageOf)
	printRunningTable(w, "Notices", "NOTICE", kindNotice, in.Conditions, ageOf)
	printConditionTable(w, inst)
}

// staleNote warns that the details printed below the header are not about
// the current spec, which the header alone would not explain.
func staleNote(in instancestatus.Input, shown instancestatus.Display) string {
	switch {
	case shown.Tone == instancestatus.ToneUnknown:
		return "No provider is reconciling this instance. Check that its provider is installed and running."
	case shown.Stale && len(in.Conditions) > 0:
		return "The latest change has not been observed yet. The details below describe the instance before it."
	default:
		return ""
	}
}

// age says how long ago t was, the way the AGE column of a list does.
func age(t, now time.Time) string {
	return duration.ShortHumanDuration(max(now.Sub(t), 0))
}

func instanceName(inst *client.Instance) string {
	if inst.Metadata == nil || inst.Metadata.Name == "" {
		return "-"
	}
	return inst.Metadata.Name
}

func printComponentTable(w io.Writer, inst *client.Instance) {
	if inst.Status == nil || inst.Status.Components == nil || len(*inst.Status.Components) == 0 {
		return
	}
	fmt.Fprintln(w, "\nComponents:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tREADY")
	for _, comp := range *inst.Status.Components {
		var ready, replicas int32
		if comp.ReadyReplicas != nil {
			ready = *comp.ReadyReplicas
		}
		if comp.Replicas != nil {
			replicas = *comp.Replicas
		}
		fmt.Fprintf(tw, "  %s\t%d/%d\n", comp.Name, ready, replicas)
	}
	tw.Flush() //nolint:errcheck,gosec
}

// conditionKind is what a condition tells a reader, which decides where the
// status shows it. The zero value covers the summary conditions, which the
// header already reads, and the types this build does not know: they appear
// only among all the conditions.
type conditionKind int

const (
	// kindCheck is True when fine, Unknown while in progress, and False
	// while a problem is observed.
	kindCheck conditionKind = iota + 1
	// kindOperation is True while a long operation runs.
	kindOperation
	// kindNotice is True when a decision is wanted and nothing is broken.
	kindNotice
)

// conditionKinds sorts the conditions an Instance can carry. A type missing
// here is not an error: it is shown as it is, with all the conditions.
//
//nolint:gochecknoglobals // immutable catalog
var conditionKinds = map[string]conditionKind{
	corev1alpha1.ConditionReferencesResolved:     kindCheck,
	corev1alpha1.ConditionSynced:                 kindCheck,
	corev1alpha1.ConditionPodsScheduled:          kindCheck,
	corev1alpha1.ConditionPodsReady:              kindCheck,
	corev1alpha1.ConditionEngineHealthy:          kindCheck,
	corev1alpha1.ConditionDataSourceReady:        kindCheck,
	corev1alpha1.ConditionBackupConfigured:       kindCheck,
	corev1alpha1.ConditionConnectionDetailsReady: kindCheck,
	corev1alpha1.ConditionMonitoringConfigured:   kindCheck,
	corev1alpha1.ConditionMonitoringReady:        kindCheck,

	corev1alpha1.ConditionRestoring:       kindOperation,
	corev1alpha1.ConditionUpgrading:       kindOperation,
	corev1alpha1.ConditionStorageResizing: kindOperation,

	corev1alpha1.ConditionMaintenancePending:         kindNotice,
	corev1alpha1.ConditionComponentVersionDeprecated: kindNotice,
}

// conditionTitle is how a person reads a condition's type. A type reads well
// once split into words ("Pods ready"), and an operation's type is also the
// status word it supplies, so it reads the same here as in the header.
func conditionTitle(conditionType string) string {
	return instancestatus.Label(conditionType)
}

// printProblemTable lists what is wrong: every check that is False, and the
// cause the Instance needs attention for when no check accounts for it, such
// as a spec the provider rejected.
func printProblemTable(w io.Writer, conditions []metav1.Condition, ageOf func(time.Time) string) {
	type problem struct {
		check string
		cause metav1.Condition
	}
	var problems []problem
	accounted := make(map[string]bool)
	for _, cond := range conditions {
		if conditionKinds[cond.Type] == kindCheck && cond.Status == metav1.ConditionFalse {
			problems = append(problems, problem{check: conditionTitle(cond.Type), cause: cond})
			accounted[cond.Reason] = true
		}
	}
	for _, cond := range conditions {
		if cond.Type == corev1alpha1.ConditionNeedsAttention && cond.Status == metav1.ConditionTrue && !accounted[cond.Reason] {
			problems = append([]problem{{check: "-", cause: cond}}, problems...)
		}
	}
	if len(problems) == 0 {
		return
	}

	fmt.Fprintln(w, "\nProblems:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  PROBLEM\tCHECK\tAGE\tMESSAGE")
	for _, p := range problems {
		writeRow(tw, p.cause.Message, instancestatus.Label(p.cause.Reason), p.check, ageOf(p.cause.LastTransitionTime.Time))
	}
	tw.Flush() //nolint:errcheck,gosec
}

// printCheckTable is the checklist: how many checks pass, how many fail and
// which are still pending. While an instance is being created this is the
// view of its progress. The failing ones are listed with the problems, and a
// check that does not apply to the instance is not in its status and so not
// here either.
func printCheckTable(w io.Writer, conditions []metav1.Condition) {
	var passing, failing int
	var pending []metav1.Condition
	for _, cond := range conditions {
		if conditionKinds[cond.Type] != kindCheck {
			continue
		}
		switch cond.Status {
		case metav1.ConditionTrue:
			passing++
		case metav1.ConditionFalse:
			failing++
		case metav1.ConditionUnknown:
			pending = append(pending, cond)
		}
	}
	total := passing + failing + len(pending)
	if total == 0 {
		return
	}

	summary := fmt.Sprintf("%d of %d passing", passing, total)
	if failing > 0 {
		summary += fmt.Sprintf(", %d failing", failing)
	}
	if len(pending) > 0 {
		summary += fmt.Sprintf(", %d pending", len(pending))
	}
	fmt.Fprintln(w, "\nChecks:")
	fmt.Fprintf(w, "  %s\n", summary)
	if len(pending) == 0 {
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  PENDING\tMESSAGE")
	for _, cond := range pending {
		writeRow(tw, cond.Message, conditionTitle(cond.Type))
	}
	tw.Flush() //nolint:errcheck,gosec
}

// printRunningTable lists the conditions of one kind that are True: the
// operations in progress, or the notices that want a decision.
func printRunningTable(w io.Writer, title, column string, kind conditionKind, conditions []metav1.Condition, ageOf func(time.Time) string) {
	var running []metav1.Condition
	for _, cond := range conditions {
		if conditionKinds[cond.Type] == kind && cond.Status == metav1.ConditionTrue {
			running = append(running, cond)
		}
	}
	if len(running) == 0 {
		return
	}

	fmt.Fprintf(w, "\n%s:\n", title)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\tAGE\tMESSAGE\n", column)
	for _, cond := range running {
		writeRow(tw, cond.Message, conditionTitle(cond.Type), ageOf(cond.LastTransitionTime.Time))
	}
	tw.Flush() //nolint:errcheck,gosec
}

func printConditionTable(w io.Writer, inst *client.Instance) {
	if inst.Status == nil || inst.Status.Conditions == nil || len(*inst.Status.Conditions) == 0 {
		return
	}
	fmt.Fprintln(w, "\nConditions:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  TYPE\tSTATUS\tREASON\tMESSAGE")
	for _, cond := range *inst.Status.Conditions {
		writeRow(tw, cond.Message, cond.Type, string(cond.Status), cond.Reason)
	}
	tw.Flush() //nolint:errcheck,gosec
}

// writeRow writes a table row that ends with a message. A message of several
// lines continues under its own column, and an empty one is a dash.
func writeRow(tw io.Writer, message string, cells ...string) {
	if message == "" {
		message = "-"
	}
	lines := strings.Split(message, "\n")
	fmt.Fprintf(tw, "  %s\t%s\n", strings.Join(cells, "\t"), lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(tw, "  %s%s\n", strings.Repeat("\t", len(cells)), line)
	}
}
