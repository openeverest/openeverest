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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// UpdateOptions configures `instance update`.
type UpdateOptions struct {
	Name       string
	Namespace  string
	Cluster    string
	Context    string   // overrides the active context when set
	ValuesFile string   // path to a YAML values file with spec-level overrides
	Set        []string // dot-notation overrides e.g. "components.engine.replicas=3"; takes precedence over ValuesFile
	DryRun     bool     // print the paths the patch names, without writing
}

// Updater implements `instance update` business logic.
type Updater struct {
	config Config
	l      *zap.SugaredLogger
}

// NewUpdater returns a new Updater.
func NewUpdater(cfg Config, l *zap.SugaredLogger) *Updater {
	iu := &Updater{config: cfg, l: l.With("component", "instance-updater")}
	if cfg.Pretty {
		iu.l = zap.NewNop().Sugar()
	}
	return iu
}

// Run sends the -f/--set overrides as a single RFC 7386 merge patch. The server
// applies it, so members the patch does not name keep their value, a member set
// to null is removed.
func (iu *Updater) Run(ctx context.Context, opts UpdateOptions, cfgPath string) error {
	if opts.ValuesFile == "" && len(opts.Set) == 0 {
		return fmt.Errorf("at least one of --set or -f/--file is required")
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: iu.config.Pretty}, iu.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	overrides, err := buildSpecOverrides(opts.ValuesFile, opts.Set)
	if err != nil {
		return err
	}

	// The write is one PATCH; this GET only serves the component-name check and
	// --dry-run's current values, so it is skipped when neither needs it.
	components := patchedComponents(overrides)
	var inst *client.Instance
	if opts.DryRun || len(components) > 0 {
		if inst, err = getInstance(ctx, c, opts.Cluster, opts.Namespace, opts.Name); err != nil {
			return err
		}
	}
	if len(components) > 0 {
		if err := iu.checkComponents(ctx, c, opts.Cluster, inst, components); err != nil {
			return err
		}
	}

	patch := map[string]any{specKey: overrides}
	if opts.DryRun {
		return iu.emitDryRun(inst, overrides, patch, opts)
	}

	updated, err := iu.patch(ctx, c, opts, patch)
	if err != nil {
		return err
	}
	return iu.emitUpdated(updated, opts)
}

// patchedComponents names the components the patch touches. spec.components is a
// map, so a misspelt name is a structurally valid member that fieldValidation=Strict
// cannot reject, unlike a misspelt field.
func patchedComponents(overrides map[string]any) []string {
	named, isObject := overrides[componentsKey].(map[string]any)
	if !isObject {
		return nil
	}
	names := make([]string, 0, len(named))
	for name := range named {
		names = append(names, name)
	}
	return names
}

// checkComponents rejects a patch naming a component the instance's provider does
// not offer, and fails open so a lookup failure cannot block a valid update.
func (iu *Updater) checkComponents(ctx context.Context, c *client.ClientWithResponses, cluster string, inst *client.Instance, names []string) error {
	provider := inst.Spec.ProviderRef.Name
	if provider == "" {
		return nil
	}

	resp, err := c.GetProviderWithResponse(ctx, cluster, provider)
	if err != nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		// Warn rather than log: iu.l is a no-op in pretty mode, which is the default,
		// so silence here would read as "your component names were checked".
		output.PrintWarn(fmt.Sprintf(
			"could not fetch provider %q to validate component names, leaving it to the server", provider,
		), iu.l, iu.config.Pretty)
		return nil //nolint:nilerr // a lookup failure must not block a valid update
	}

	var topology string
	if inst.Spec.Topology != nil && inst.Spec.Topology.Type != nil {
		topology = *inst.Spec.Topology.Type
	}
	return validateComponentNames(names, resp.JSON200, topology)
}

func getInstance(ctx context.Context, c *client.ClientWithResponses, cluster, namespace, name string) (*client.Instance, error) {
	resp, err := c.GetInstanceWithResponse(ctx, cluster, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance %q: %w", name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("instance %q not found in namespace %q", name, namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response fetching instance %q: %s", name, resp.Status())
	}
	return resp.JSON200, nil
}

func (iu *Updater) patch(ctx context.Context, c *client.ClientWithResponses, opts UpdateOptions, patch map[string]any) (*client.Instance, error) {
	resp, err := c.PatchInstanceWithApplicationMergePatchPlusJSONBodyWithResponse(
		ctx, opts.Cluster, opts.Namespace, opts.Name, patch,
	)
	if err != nil {
		return nil, fmt.Errorf("patch instance request failed: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return resp.JSON200, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("instance %q not found in namespace %q", opts.Name, opts.Namespace)
	default:
		if msg, ok := clienterr.Message(resp.JSON415, resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response updating instance %q: %s", opts.Name, resp.Status())
	}
}

type specChange struct {
	path string
	from string
	to   string
}

// changedPaths pairs every leaf of the patch with the instance's current value at
// the same path. A list is a leaf: a merge patch replaces one wholesale.
func changedPaths(current, patch map[string]any, prefix string) []specChange {
	keys := make([]string, 0, len(patch))
	for k := range patch {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var changes []specChange
	for _, k := range keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		cur, present := current[k]
		if child, isObject := patch[k].(map[string]any); isObject {
			curChild, _ := cur.(map[string]any) // a missing or scalar current reads as empty
			changes = append(changes, changedPaths(curChild, child, path)...)
			continue
		}
		changes = append(changes, specChange{path: path, from: formatValue(cur, present), to: formatValue(patch[k], true)})
	}
	return changes
}

func formatValue(v any, present bool) string {
	if !present {
		return "<unset>"
	}
	switch v.(type) {
	case nil:
		return nullValue
	case map[string]any, []any:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return fmt.Sprintf("%v", v)
}

// specToMap round-trips a typed spec through JSON so it can be walked by path.
func specToMap(spec any) (map[string]any, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}
	return m, nil
}

// emitDryRun lists the paths the patch names with their current and new values,
// which is a patch's whole contract. JSON mode emits the patch document itself.
func (iu *Updater) emitDryRun(inst *client.Instance, overrides, patch map[string]any, opts UpdateOptions) error {
	if !iu.config.Pretty {
		if err := json.NewEncoder(os.Stdout).Encode(patch); err != nil {
			return fmt.Errorf("failed to encode patch: %w", err)
		}
		return nil
	}

	current, err := specToMap(inst.Spec)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(os.Stdout, output.Info("Dry run for instance %q, no changes written", opts.Name))
	_, _ = fmt.Fprint(os.Stdout, renderChanges(changedPaths(current, overrides, "")))
	return nil
}

// renderChanges lists every path the patch names, labelling the ones already at
// the value being set rather than printing them as "3 -> 3" or hiding them: a
// --set that does nothing is the thing the caller most needs told. Mirrors how
// `kubectl patch` reports "patched (no change)".
func renderChanges(changes []specChange) string {
	var b strings.Builder
	changed := 0
	for _, ch := range changes {
		if ch.from == ch.to {
			fmt.Fprintf(&b, "  %s: %s (no change)\n", ch.path, ch.to)
			continue
		}
		changed++
		fmt.Fprintf(&b, "  %s: %s -> %s\n", ch.path, ch.from, ch.to)
	}
	if changed == 0 {
		b.WriteString(output.Warn("Nothing would change"))
	}
	return b.String()
}

func (iu *Updater) emitUpdated(updated *client.Instance, opts UpdateOptions) error {
	if iu.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q updated in namespace %q", opts.Name, opts.Namespace))
		return nil
	}
	if updated == nil {
		return fmt.Errorf("instance %q was updated but the server returned an unreadable response body", opts.Name)
	}
	return writeInstanceJSON(updated)
}
