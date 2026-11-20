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

import { ClusterInstancePreset, InstancePreset } from 'shared-types/api.types';
import { PRESET_ANNOTATION_BY_SCOPE } from './preset-selection.constants';
import {
  PresetIdParts,
  PresetScope,
  SelectablePreset,
} from './preset-selection.types';

// Kubernetes names never contain "/", so it is a safe separator. A namespace
// preset's id carries its namespace: after a namespace switch the picked id no
// longer matches anything, instead of silently resolving to a same-named
// preset with a different spec.
const ID_SEPARATOR = '/';

export const getClusterPresetId = (name: string): string =>
  [PresetScope.Cluster, name].join(ID_SEPARATOR);

export const getNamespacePresetId = (namespace: string, name: string): string =>
  [PresetScope.Namespace, namespace, name].join(ID_SEPARATOR);

export const parsePresetId = (id: string): PresetIdParts | undefined => {
  const [scope, ...rest] = id.split(ID_SEPARATOR);
  if (scope === PresetScope.Cluster && rest.length === 1) {
    return { scope, name: rest[0] };
  }
  if (scope === PresetScope.Namespace && rest.length === 2) {
    return { scope, namespace: rest[0], name: rest[1] };
  }
  return undefined;
};

const toSelectablePresets = (
  presets: (ClusterInstancePreset | InstancePreset)[],
  scope: PresetScope,
  getId: (name: string) => string
): SelectablePreset[] =>
  presets.flatMap(({ metadata, spec }) =>
    metadata?.name
      ? [{ id: getId(metadata.name), scope, name: metadata.name, spec }]
      : []
  );

// Cluster presets first: they are known before a namespace is, so namespace
// presets are appended without moving the cards already on screen. The scope
// comes from the endpoint each list was read from, never from the items.
export const mergePresets = (
  clusterPresets: ClusterInstancePreset[] = [],
  namespacePresets: InstancePreset[] = [],
  namespace: string
): SelectablePreset[] => [
  ...toSelectablePresets(
    clusterPresets,
    PresetScope.Cluster,
    getClusterPresetId
  ),
  ...toSelectablePresets(namespacePresets, PresetScope.Namespace, (name) =>
    getNamespacePresetId(namespace, name)
  ),
];

interface BuildPresetCreateArgsInput {
  selectedPreset: SelectablePreset;
  provider: string;
  dbName: string;
  k8sNamespace: string;
}

interface PresetCreateArgs {
  formValue: {
    provider: string;
    dbName: string;
    k8sNamespace: string;
    spec: SelectablePreset['spec'];
  };
  annotations: Record<string, string>;
}

// Builds the create-instance arguments for a one-click preset deploy: the
// selected preset spec is sent verbatim (never rebuilt from the wizard fields)
// and the originating preset is recorded in the annotation of its kind. Only
// request-address fields ride alongside the spec; they are stripped before
// submit by buildCreateInstanceSpec, so no wizard-only value leaks into the
// Instance spec.
export const buildPresetCreateArgs = ({
  selectedPreset,
  provider,
  dbName,
  k8sNamespace,
}: BuildPresetCreateArgsInput): PresetCreateArgs => ({
  formValue: {
    provider,
    dbName,
    k8sNamespace,
    spec: selectedPreset.spec,
  },
  annotations: {
    [PRESET_ANNOTATION_BY_SCOPE[selectedPreset.scope]]: selectedPreset.name,
  },
});
