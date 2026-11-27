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

package preset

import (
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/client"
)

// loadPresetFile reads an InstancePreset manifest from path (YAML or JSON).
func loadPresetFile(path string) (*client.InstancePreset, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a user-supplied CLI flag
	if err != nil {
		return nil, fmt.Errorf("failed to read preset file %q: %w", path, err)
	}
	var preset client.InstancePreset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("failed to parse preset file %q: %w", path, err)
	}
	return &preset, nil
}

// applyIdentity sets metadata.name / metadata.namespace from flags when given,
// and returns the resolved name and namespace. Either flag or the file must
// supply both; flags win over the file.
func applyIdentity(preset *client.InstancePreset, name, namespace string) (string, string, error) {
	if preset.Metadata == nil {
		preset.Metadata = &client.ObjectMeta{}
	}
	if name != "" {
		preset.Metadata.Name = name
	}
	if namespace != "" {
		preset.Metadata.Namespace = namespace
	}
	resolvedName := preset.Metadata.Name
	resolvedNamespace := preset.Metadata.Namespace
	if resolvedName == "" {
		return "", "", fmt.Errorf("preset name is required: pass --name or set metadata.name in the file")
	}
	if resolvedNamespace == "" {
		return "", "", fmt.Errorf("preset namespace is required: pass --namespace or set metadata.namespace in the file")
	}
	return resolvedName, resolvedNamespace, nil
}

// stripServerFields clears fields that must not be sent on create, and that
// would conflict with a PUT update when taken from a get -o yaml export.
func stripServerFields(preset *client.InstancePreset) {
	preset.Status = nil
	if preset.Metadata == nil {
		return
	}
	preset.Metadata.UID = ""
	preset.Metadata.ResourceVersion = ""
	preset.Metadata.Generation = 0
	preset.Metadata.CreationTimestamp = metav1.Time{}
	preset.Metadata.ManagedFields = nil
	// Clear legacy selfLink if present in a get -o yaml export.
	preset.Metadata.SelfLink = "" //nolint:staticcheck // SA1019: field is deprecated but still appears in older exports
}
