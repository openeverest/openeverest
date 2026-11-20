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

import { buildCreateInstanceSpec } from 'hooks/api/db-instances/useCreateDbInstance';
import { ClusterInstancePreset, InstancePreset } from 'shared-types/api.types';
import { PresetScope, SelectablePreset } from './preset-selection.types';
import {
  buildPresetCreateArgs,
  getClusterPresetId,
  getNamespacePresetId,
  mergePresets,
  parsePresetId,
} from './preset-selection.utils';

const spec: SelectablePreset['spec'] = {
  providerRef: { name: 'psmdb' },
  version: '8.0.12',
  topology: { type: 'replicaSet' },
  components: { engine: { replicas: 3 } },
};

const clusterPreset: SelectablePreset = {
  id: getClusterPresetId('psmdb-replicaset'),
  scope: PresetScope.Cluster,
  name: 'psmdb-replicaset',
  spec,
};

const namespacePreset: SelectablePreset = {
  id: getNamespacePresetId('prod', 'psmdb-replicaset'),
  scope: PresetScope.Namespace,
  name: 'psmdb-replicaset',
  spec,
};

const createArgs = (selectedPreset: SelectablePreset) =>
  buildPresetCreateArgs({
    selectedPreset,
    provider: 'psmdb',
    dbName: 'my-db',
    k8sNamespace: 'prod',
  });

describe('buildPresetCreateArgs', () => {
  it('sends the selected preset spec verbatim (same reference)', () => {
    expect(createArgs(clusterPreset).formValue.spec).toBe(spec);
  });

  // The server looks the preset up by the kind the annotation names, and
  // refuses an instance carrying both: exactly one, matching the scope.
  it('records a cluster preset in the cluster annotation only', () => {
    expect(createArgs(clusterPreset).annotations).toEqual({
      'openeverest.io/cluster-instance-preset': 'psmdb-replicaset',
    });
  });

  it('records a namespace preset in the namespace annotation only', () => {
    expect(createArgs(namespacePreset).annotations).toEqual({
      'openeverest.io/instance-preset': 'psmdb-replicaset',
    });
  });

  it('produces a spec of providerRef + preset spec, with no wizard meta leaking', () => {
    const instanceSpec = buildCreateInstanceSpec({
      ...createArgs(namespacePreset).formValue,
      presetId: namespacePreset.id,
    });

    expect(instanceSpec).toEqual({ ...spec, providerRef: { name: 'psmdb' } });
    // Request-address / UI-only fields must never end up in the Instance spec.
    expect(instanceSpec).not.toHaveProperty('dbName');
    expect(instanceSpec).not.toHaveProperty('k8sNamespace');
    expect(instanceSpec).not.toHaveProperty('presetId');
    expect(instanceSpec).not.toHaveProperty('provider');
  });
});

describe('mergePresets', () => {
  const clusterItem: ClusterInstancePreset = {
    metadata: { name: 'psmdb-default' },
    spec,
  };
  const namespaceItem: InstancePreset = {
    metadata: { name: 'psmdb-default', namespace: 'prod' },
    spec,
  };

  it('gives same-named cluster and namespace presets distinct ids', () => {
    const [first, second] = mergePresets(
      [clusterItem],
      [namespaceItem],
      'prod'
    );

    expect(first.scope).toBe(PresetScope.Cluster);
    expect(second.scope).toBe(PresetScope.Namespace);
    expect(first.id).not.toBe(second.id);
    expect(first.name).toBe(second.name);
  });

  it('gives a same-named preset a different id in another namespace', () => {
    const [inProd] = mergePresets([], [namespaceItem], 'prod');
    const [inStaging] = mergePresets([], [namespaceItem], 'staging');

    expect(inProd.id).not.toBe(inStaging.id);
  });

  it('skips items that cannot be referenced by name', () => {
    expect(mergePresets([{ spec }], [{ metadata: {}, spec }], 'prod')).toEqual(
      []
    );
  });
});

describe('parsePresetId', () => {
  it('reads back what the id builders wrote', () => {
    expect(parsePresetId(getClusterPresetId('psmdb-default'))).toEqual({
      scope: PresetScope.Cluster,
      name: 'psmdb-default',
    });
    expect(
      parsePresetId(getNamespacePresetId('prod', 'psmdb-default'))
    ).toEqual({
      scope: PresetScope.Namespace,
      namespace: 'prod',
      name: 'psmdb-default',
    });
  });

  it('rejects an id it did not build', () => {
    expect(parsePresetId('psmdb-default')).toBeUndefined();
    expect(parsePresetId('namespace/psmdb-default')).toBeUndefined();
  });
});
