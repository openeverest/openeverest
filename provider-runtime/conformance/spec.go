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

package conformance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// loadProviderSpec reads the generated Provider CR — the artifact that ships in
// the Helm chart and that the UI actually consumes.
func loadProviderSpec(path string) (*corev1alpha1.ProviderSpec, error) {
	if path == "" {
		located, err := findProviderSpec()
		if err != nil {
			return nil, err
		}
		path = located
	}

	data, err := os.ReadFile(path) //nolint:gosec // a path inside the module under test
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc struct {
		Spec corev1alpha1.ProviderSpec `json:"spec"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Spec.Topologies) == 0 {
		return nil, fmt.Errorf("%s declares no topologies; run `make generate` first", path)
	}
	return &doc.Spec, nil
}

// findProviderSpec walks up to the module root and looks for the generated
// chart artifact, so callers do not have to hardcode a path relative to
// whichever package their test happens to live in.
func findProviderSpec() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find the module root; set Config.ProviderSpecPath")
		}
		dir = parent
	}

	matches, err := filepath.Glob(filepath.Join(dir, "charts", "*", "generated", "provider-spec.yaml"))
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no charts/*/generated/provider-spec.yaml under %s; run `make generate` first", dir)
	default:
		return "", fmt.Errorf("found %d generated provider specs under %s; set Config.ProviderSpecPath", len(matches), dir)
	}
}

// uiPaths returns the Instance fields the topology's UI schema binds form
// controls to. Only `path` values count: a field merely referenced by a CEL
// expression is read by the form, not written by it.
func uiPaths(spec *corev1alpha1.ProviderSpec, topology string) ([]string, error) {
	if spec.UISchema == nil || len(spec.UISchema.Raw) == 0 {
		return nil, nil
	}

	var schemas map[string]any
	if err := yaml.Unmarshal(spec.UISchema.Raw, &schemas); err != nil {
		return nil, err
	}

	var paths []string
	collectPaths(schemas[topology], &paths)
	return dedupe(paths), nil
}

func collectPaths(node any, out *[]string) {
	const pathKey = "path"

	switch t := node.(type) {
	case map[string]any:
		for key, value := range t {
			if key == pathKey {
				if s, ok := value.(string); ok {
					*out = append(*out, s)
				}
				continue
			}
			collectPaths(value, out)
		}
	case []any:
		for _, value := range t {
			collectPaths(value, out)
		}
	}
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// providerObject rebuilds the Provider CR from the generated spec so providers
// that resolve images or schemas from it behave as they do in a cluster.
func providerObject(name string, spec *corev1alpha1.ProviderSpec) *corev1alpha1.Provider {
	return &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       *spec,
	}
}
