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

// Package instance provides CLI business logic for instance management.
package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

type Config struct {
	Pretty bool
}

type CreateOptions struct {
	Name       string
	Namespace  string
	Provider   string
	Preset     string // InstancePreset name; provider must match --provider
	Cluster    string
	Version    string
	Topology   string
	Context    string        // overrides the active context when set
	ValuesFile string        // path to a YAML values file with spec-level overrides (optional)
	Set        []string      // dot-notation overrides e.g. "components.engine.replicas=3"; takes precedence over ValuesFile
	Wait       bool          // block until the instance reaches the Ready phase
	Timeout    time.Duration // bounds --wait; must be positive
}

type InstanceCreator struct {
	config Config
	l      *zap.SugaredLogger
}

func NewInstanceCreator(cfg Config, l *zap.SugaredLogger) *InstanceCreator {
	ic := &InstanceCreator{config: cfg, l: l.With("component", "instance-creator")}
	if cfg.Pretty {
		ic.l = zap.NewNop().Sugar()
	}
	return ic
}

func (ic *InstanceCreator) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	if opts.Provider == "" && opts.Preset == "" {
		return fmt.Errorf("--provider is required when --preset is not given")
	}
	if opts.Preset != "" && opts.Topology != "" {
		return fmt.Errorf("--topology cannot be combined with --preset")
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: ic.config.Pretty}, ic.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	var (
		presetSpecBase map[string]any
		annotations    map[string]string
	)
	if opts.Preset != "" {
		pr, err := ic.resolvePreset(ctx, c, opts.Preset, opts.Cluster, opts.Namespace)
		if err != nil {
			return err
		}
		if err := applyPresetDefaults(opts.Preset, &opts, pr); err != nil {
			return err
		}
		presetSpecBase = pr.specMap
		annotations = map[string]string{"openeverest.io/instance-preset": opts.Preset}
	}

	provResp, err := c.GetProviderWithResponse(ctx, opts.Cluster, opts.Provider)
	if err != nil {
		return fmt.Errorf("failed to fetch provider %q: %w", opts.Provider, err)
	}

	if provResp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("provider %q not found in cluster %q", opts.Provider, opts.Cluster)
	}

	if provResp.StatusCode() != http.StatusOK || provResp.JSON200 == nil {
		return fmt.Errorf("unexpected response fetching provider %q: %s", opts.Provider, provResp.Status())
	}

	prov := provResp.JSON200

	resolvedVersion := opts.Version
	if resolvedVersion == "" {
		resolvedVersion = defaultVersion(prov)
	}

	resolvedTopology := opts.Topology
	if resolvedTopology == "" {
		resolvedTopology = firstTopology(prov)
	} else if err := validateTopology(resolvedTopology, prov); err != nil {
		return err
	}

	if len(opts.Set) > 0 {
		if err := validateComponents(opts.Set, prov, resolvedTopology); err != nil {
			return err
		}
	}

	// Merge order: preset < -f file < --set flags.
	specOverrides, err := buildSpecOverrides(opts.ValuesFile, opts.Set)
	if err != nil {
		return err
	}
	if presetSpecBase != nil {
		if specOverrides != nil {
			deepMerge(presetSpecBase, specOverrides)
		}
		specOverrides = presetSpecBase
	}

	payload := buildPayload(opts.Name, opts.Provider, resolvedVersion, resolvedTopology, specOverrides, annotations)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize instance spec: %w", err)
	}

	resp, err := c.CreateInstanceWithBodyWithResponse(
		ctx,
		opts.Cluster,
		opts.Namespace,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create instance request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		if resp.StatusCode() == http.StatusConflict {
			return fmt.Errorf("instance %q already exists in namespace %q", opts.Name, opts.Namespace)
		}
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response creating instance: %s", resp.Status())
	}

	ic.l.Infof("created instance %q in namespace %q", opts.Name, opts.Namespace)

	// The API returns the accepted Instance as JSON201 (or JSON200).
	created := resp.JSON201
	if created == nil {
		created = resp.JSON200
	}

	if !opts.Wait {
		return ic.emitCreated(created, opts)
	}
	return ic.waitForInstance(ctx, c, created, opts)
}

// emitCreated reports a non-waiting create: a success line in pretty mode, or
// the created instance in JSON mode (so `create --json` is parseable either way).
func (ic *InstanceCreator) emitCreated(created *client.Instance, opts CreateOptions) error {
	if ic.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q created in namespace %q", opts.Name, opts.Namespace))
		return nil
	}
	// A 200/201 with an unparseable body would otherwise emit empty stdout.
	if created == nil {
		return fmt.Errorf("instance %q was created but the server returned an unreadable response body", opts.Name)
	}
	return writeInstanceJSON(created)
}

// waitForInstance blocks until the instance reaches a terminal phase. Pretty
// mode streams progress then the final status; JSON mode emits one final object.
func (ic *InstanceCreator) waitForInstance(
	ctx context.Context,
	c *client.ClientWithResponses,
	created *client.Instance,
	opts CreateOptions,
) error {
	// Keep the latest instance seen, for the final output.
	var latest *client.Instance
	basePoll := newInstancePoll(c, opts.Cluster, opts.Namespace, opts.Name)
	poll := func(ctx context.Context) (*client.Instance, error) {
		inst, err := basePoll(ctx)
		if err == nil {
			latest = inst
		}
		return inst, err
	}

	var onUpdate func(string)
	if ic.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Instance %q created; waiting for it to become ready...", opts.Name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, instanceCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { ic.l.Warnf("%v — retrying", err) },
	}); err != nil {
		return err
	}

	final := latest
	if final == nil {
		final = created
	}

	if ic.config.Pretty {
		var buf bytes.Buffer
		printInstanceStatus(&buf, final, opts.Namespace)
		_, _ = fmt.Fprint(os.Stdout, buf.String())
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q is ready", opts.Name))
		return nil
	}
	return writeInstanceJSON(final)
}

func writeInstanceJSON(inst *client.Instance) error {
	if inst == nil {
		return nil
	}
	if err := json.NewEncoder(os.Stdout).Encode(inst); err != nil {
		return fmt.Errorf("failed to encode instance: %w", err)
	}
	return nil
}

// applyPresetDefaults copies provider/version/topology from the preset into opts,
// validating that an explicit --provider matches the preset's provider.
func applyPresetDefaults(preset string, opts *CreateOptions, pr *presetResult) error {
	if pr.provider != "" && opts.Provider != "" && pr.provider != opts.Provider {
		return fmt.Errorf("--provider %q does not match preset %q provider %q", opts.Provider, preset, pr.provider)
	}
	if opts.Provider == "" {
		if pr.provider == "" {
			return fmt.Errorf("preset %q does not specify a provider; use --provider to set one", preset)
		}
		opts.Provider = pr.provider
	}
	if opts.Version == "" {
		opts.Version = pr.version
	}
	if pr.topology != "" {
		opts.Topology = pr.topology
	}
	return nil
}

type presetResult struct {
	specMap  map[string]any
	provider string
	version  string
	topology string
}

func (ic *InstanceCreator) resolvePreset(ctx context.Context, c *client.ClientWithResponses, preset, cluster, namespace string) (*presetResult, error) {
	resp, err := c.ResolveInstancePresetWithResponse(
		ctx,
		cluster,
		preset,
		&client.ResolveInstancePresetParams{Namespace: namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve preset %q: %w", preset, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("preset %q not found in cluster %q", preset, cluster)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response resolving preset %q: %s", preset, resp.Status())
	}

	p := resp.JSON200
	pr := &presetResult{}
	pr.provider = p.Spec.ProviderRef.Name
	if p.Spec.Version != nil {
		pr.version = *p.Spec.Version
	}
	if p.Spec.Topology != nil && p.Spec.Topology.Type != nil {
		pr.topology = *p.Spec.Topology.Type
	}
	specMap, err := presetSpecToMap(p)
	if err != nil {
		return nil, err
	}
	pr.specMap = specMap
	return pr, nil
}

func presetSpecToMap(p *client.InstancePreset) (map[string]any, error) {
	b, err := json.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preset spec: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("failed to parse preset spec: %w", err)
	}
	return m, nil
}

func defaultVersion(prov *client.Provider) string {
	if prov.Spec.Versions == nil {
		return ""
	}

	versions := *prov.Spec.Versions
	first := ""
	for _, v := range versions {
		if first == "" {
			first = v.Name
		}
		if v.Default != nil && *v.Default {
			return v.Name
		}
	}
	return first
}

func firstTopology(prov *client.Provider) string {
	if prov.Spec.Topologies == nil {
		return ""
	}

	keys := make([]string, 0, len(*prov.Spec.Topologies))
	for k := range *prov.Spec.Topologies {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		return ""
	}

	sort.Strings(keys)
	return keys[0]
}

func validateTopology(topology string, prov *client.Provider) error {
	if prov.Spec.Topologies == nil {
		return nil
	}
	if _, ok := (*prov.Spec.Topologies)[topology]; !ok {
		names := make([]string, 0, len(*prov.Spec.Topologies))
		for k := range *prov.Spec.Topologies {
			names = append(names, k)
		}
		sort.Strings(names)
		return fmt.Errorf(
			"topology %q is not available for provider %q\nvalid topologies: %s",
			topology, providerName(prov), strings.Join(names, ", "),
		)
	}
	return nil
}

// validateComponents only checks --set paths starting with "components.".
func validateComponents(setFlags []string, prov *client.Provider, topology string) error {
	// Topology components are the ground truth, but the API strips null entries,
	// so fall back to spec.components (global registry) when the topology map is empty.
	valid := map[string]struct{}{}
	if prov.Spec.Topologies != nil {
		if t, ok := (*prov.Spec.Topologies)[topology]; ok && t.Components != nil {
			for name := range *t.Components {
				valid[name] = struct{}{}
			}
		}
	}
	if len(valid) == 0 && prov.Spec.Components != nil {
		for name := range *prov.Spec.Components {
			valid[name] = struct{}{}
		}
	}
	if len(valid) == 0 {
		return nil // can't determine valid names; let the server validate
	}

	var invalid []string
	seen := map[string]struct{}{}
	for _, s := range setFlags {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) < 2 {
			continue
		}
		segments := strings.SplitN(parts[0], ".", 3)
		if len(segments) < 2 || segments[0] != "components" {
			continue
		}
		compName := segments[1]
		if _, done := seen[compName]; done {
			continue
		}
		seen[compName] = struct{}{}
		if _, ok := valid[compName]; !ok {
			invalid = append(invalid, compName)
		}
	}

	if len(invalid) == 0 {
		return nil
	}

	validNames := make([]string, 0, len(valid))
	for name := range valid {
		validNames = append(validNames, name)
	}
	sort.Strings(validNames)
	sort.Strings(invalid)

	return fmt.Errorf(
		"invalid component(s) for provider %q with topology %q: %s\nvalid components: %s",
		providerName(prov),
		topology,
		strings.Join(invalid, ", "),
		strings.Join(validNames, ", "),
	)
}

func providerName(prov *client.Provider) string {
	if prov.Metadata == nil || prov.Metadata.Name == "" {
		return "<unknown>"
	}
	return prov.Metadata.Name
}

// buildSpecOverrides merges -f file values and --set overrides; --set wins.
func buildSpecOverrides(valuesFile string, setFlags []string) (map[string]any, error) {
	var base map[string]any

	if valuesFile != "" {
		loaded, err := loadValuesFile(valuesFile)
		if err != nil {
			return nil, err
		}
		base = loaded
	}

	overrides, err := parseSetFlags(setFlags)
	if err != nil {
		return nil, err
	}

	if base == nil {
		return overrides, nil
	}
	if overrides == nil {
		return base, nil
	}

	deepMerge(base, overrides)
	return base, nil
}

func loadValuesFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read values file %q: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("cannot parse values file %q: %w", path, err)
	}

	return raw, nil
}

// deepMerge merges src into dst; src scalars overwrite dst, maps recurse.
func deepMerge(dst, src map[string]any) {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		srcMap, srcIsMap := sv.(map[string]any)
		dstMap, dstIsMap := dv.(map[string]any)
		if srcIsMap && dstIsMap {
			deepMerge(dstMap, srcMap)
		} else {
			dst[k] = sv
		}
	}
}

// parseSetFlags parses "field.sub=value" entries into a nested map.
func parseSetFlags(setFlags []string) (map[string]any, error) {
	if len(setFlags) == 0 {
		return nil, nil
	}

	result := map[string]any{}

	for _, s := range setFlags {
		eqIdx := strings.Index(s, "=")
		if eqIdx < 0 {
			return nil, fmt.Errorf("invalid --set value %q: must be in the form field.subfield=value", s)
		}

		path := s[:eqIdx]
		rawValue := s[eqIdx+1:]

		if path == "" {
			return nil, fmt.Errorf("invalid --set flag %q: path must not be empty", s)
		}

		segments := strings.Split(path, ".")
		value := coerceValue(rawValue)
		if err := deepSet(result, segments, value); err != nil {
			return nil, fmt.Errorf("conflicting --set paths at %q: %w", path, err)
		}
	}

	return result, nil
}

func coerceValue(s string) any {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}

func deepSet(m map[string]any, path []string, value any) error {
	if len(path) == 1 {
		m[path[0]] = value
		return nil
	}

	child, ok := m[path[0]]
	if !ok {
		child = map[string]any{}
		m[path[0]] = child
	}

	childMap, ok := child.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot set sub-field of scalar value at %q", path[0])
	}

	return deepSet(childMap, path[1:], value)
}

// buildPayload builds the Instance JSON payload; explicit flags win over --set/-f.
func buildPayload(name, provider, version, topology string, specOverrides map[string]any, annotations map[string]string) map[string]any {
	if specOverrides == nil {
		specOverrides = map[string]any{}
	}

	specOverrides["providerRef"] = map[string]any{"name": provider}
	if version != "" {
		specOverrides["version"] = version
	}
	if topology != "" {
		// Merge type into existing topology object to preserve topology.config from preset.
		if existing, ok := specOverrides["topology"].(map[string]any); ok {
			existing["type"] = topology
		} else {
			specOverrides["topology"] = map[string]any{"type": topology}
		}
	}

	metadata := map[string]any{"name": name}
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}

	return map[string]any{
		"metadata": metadata,
		"spec":     specOverrides,
	}
}
