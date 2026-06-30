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
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

type Config struct {
	Pretty bool
}

type CreateOptions struct {
	Name       string
	Namespace  string
	Provider   string
	Cluster    string
	Version    string
	Topology   string
	Context    string   // overrides the active context when set
	ValuesFile string   // path to a YAML file with spec-level overrides (optional)
	Set        []string // each entry: "specField.subfield=value" e.g. "components.engine.replicas=3" — takes precedence over ValuesFile
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
	sess, err := cli.LoadSession(cfgPath, opts.Context)
	if err != nil {
		return err
	}

	// Refresh proactively within 30s of expiry to avoid a mid-flight 401.
	if time.Now().After(sess.User.ExpiresAt.Add(-30 * time.Second)) {
		lo := authcli.NewLogin(authcli.Config{Pretty: ic.config.Pretty}, ic.l.Desugar().Sugar())
		if err := lo.Refresh(ctx, cfgPath); err != nil {
			return fmt.Errorf("access token expired and refresh failed: %w", err)
		}
		sess, err = cli.LoadSession(cfgPath, opts.Context)
		if err != nil {
			return err
		}
	}

	c, err := client.NewClientWithResponses(cli.NormalizeServerURL(sess.Server.URL))
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	token := cli.BearerToken(sess.User.AccessToken)

	provResp, err := c.GetProviderWithResponse(ctx, opts.Cluster, opts.Provider, token)
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

	specOverrides, err := buildSpecOverrides(opts.ValuesFile, opts.Set)
	if err != nil {
		return err
	}

	payload := buildPayload(opts.Name, opts.Provider, resolvedVersion, resolvedTopology, specOverrides)

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
		token,
	)
	if err != nil {
		return fmt.Errorf("create instance request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		if resp.StatusCode() == http.StatusConflict {
			return fmt.Errorf("instance %q already exists in namespace %q", opts.Name, opts.Namespace)
		}
		if resp.JSONDefault != nil && resp.JSONDefault.Message != nil {
			return fmt.Errorf("server error: %s", *resp.JSONDefault.Message)
		}
		return fmt.Errorf("unexpected response creating instance: %s", resp.Status())
	}

	ic.l.Infof("created instance %q in namespace %q", opts.Name, opts.Namespace)
	if ic.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q created in namespace %q", opts.Name, opts.Namespace))
	}

	return nil
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
	if prov.Metadata == nil {
		return "<unknown>"
	}
	if name, ok := (*prov.Metadata)["name"]; ok {
		if s, ok := name.(string); ok {
			return s
		}
	}
	return "<unknown>"
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
func buildPayload(name, provider, version, topology string, specOverrides map[string]any) map[string]any {
	if specOverrides == nil {
		specOverrides = map[string]any{}
	}

	specOverrides["provider"] = provider
	if version != "" {
		specOverrides["version"] = version
	}
	if topology != "" {
		specOverrides["topology"] = map[string]any{"type": topology}
	}

	return map[string]any{
		"metadata": map[string]any{"name": name},
		"spec":     specOverrides,
	}
}

