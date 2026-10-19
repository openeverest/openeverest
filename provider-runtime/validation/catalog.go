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

package validation

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// checkCatalog checks the topology, components and versions an Instance names
// against the Provider. Without a topology type the topology's component set
// is not checked, because the provider picks its default topology itself.
func checkCatalog(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) (field.ErrorList, []string) {
	var errs field.ErrorList
	specPath := field.NewPath("spec")

	if v := in.Spec.Version; v != "" && !slices.ContainsFunc(spec.Versions, func(b v1alpha1.VersionBundle) bool {
		return b.Name == v
	}) {
		errs = append(errs, field.NotSupported(specPath.Child("version"), v, bundleNames(spec)))
	}

	topologyName := in.GetTopologyType()
	var topology *v1alpha1.Topology
	if t, ok := spec.Topologies[topologyName]; ok {
		topology = &t
	} else if topologyName != "" {
		errs = append(errs, field.NotSupported(
			specPath.Child("topology", "type"), topologyName, slices.Sorted(maps.Keys(spec.Topologies)),
		))
	}

	var warnings []string
	componentsPath := specPath.Child("components")
	for _, name := range slices.Sorted(maps.Keys(in.Spec.Components)) {
		componentErrs, warning := checkComponent(componentsPath.Key(name), name, in.Spec.Components[name], spec, topologyName, topology)
		errs = append(errs, componentErrs...)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}

	if topology != nil {
		for _, name := range slices.Sorted(maps.Keys(topology.Components)) {
			if _, ok := in.Spec.Components[name]; !ok && !topology.Components[name].Optional {
				errs = append(errs, field.Required(
					componentsPath.Key(name), fmt.Sprintf("the %s topology requires this component", topologyName),
				))
			}
		}
	}
	return errs, warnings
}

// checkComponent checks one component against the Provider and, when the
// topology is known, against the topology's component set.
func checkComponent(
	path *field.Path,
	name string,
	component v1alpha1.ComponentSpec,
	spec *v1alpha1.ProviderSpec,
	topologyName string,
	topology *v1alpha1.Topology,
) (field.ErrorList, string) {
	declared, ok := spec.Components[name]
	if !ok {
		return field.ErrorList{field.NotSupported(path, name, slices.Sorted(maps.Keys(spec.Components)))}, ""
	}
	if topology != nil {
		if _, ok := topology.Components[name]; !ok {
			return field.ErrorList{field.Forbidden(path, fmt.Sprintf(
				"not a component of the %s topology, which has %s",
				topologyName, strings.Join(slices.Sorted(maps.Keys(topology.Components)), ", "),
			))}, ""
		}
	}

	var errs field.ErrorList
	if component.Type != "" && component.Type != declared.Type {
		errs = append(errs, field.NotSupported(path.Child("type"), component.Type, []string{declared.Type}))
	}
	if component.Version == "" {
		return errs, ""
	}
	err, warning := checkComponentVersion(path.Child("version"), component.Version, spec.ComponentTypes[declared.Type])
	if err != nil {
		errs = append(errs, err)
	}
	return errs, warning
}

// checkComponentVersion returns an error when version is not in the
// component type's catalog, and a warning when the catalog deprecates it.
func checkComponentVersion(path *field.Path, version string, ct v1alpha1.ComponentType) (*field.Error, string) {
	known := make([]string, 0, len(ct.Versions))
	for _, v := range ct.Versions {
		if v.Version != version {
			known = append(known, v.Version)
			continue
		}
		if !v.Deprecated {
			return nil, ""
		}
		warning := fmt.Sprintf("%s: version %s is deprecated", path, version)
		if v.RemovedInVersion != "" {
			warning += fmt.Sprintf(" and is removed in provider release %s", v.RemovedInVersion)
		}
		return nil, warning
	}
	return field.NotSupported(path, version, known), ""
}

func bundleNames(spec *v1alpha1.ProviderSpec) []string {
	names := make([]string, 0, len(spec.Versions))
	for _, b := range spec.Versions {
		names = append(names, b.Name)
	}
	return names
}
