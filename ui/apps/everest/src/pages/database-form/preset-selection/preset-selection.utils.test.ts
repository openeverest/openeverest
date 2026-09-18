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
import { InstancePreset } from 'shared-types/api.types';
import { buildPresetCreateArgs } from './preset-selection.utils';
import { INSTANCE_PRESET_ANNOTATION } from './preset-selection.constants';

const resolvedPreset = {
  spec: {
    version: '8.0.12',
    topology: { type: 'replicaSet' },
    components: { engine: { replicas: 3 } },
  },
} as unknown as InstancePreset;

describe('buildPresetCreateArgs', () => {
  it('sends the resolved preset spec verbatim (same reference)', () => {
    const args = buildPresetCreateArgs({
      resolvedPreset,
      provider: 'psmdb',
      dbName: 'my-db',
      k8sNamespace: 'prod',
      presetName: 'psmdb-replicaset',
    });

    expect(args.formValue.spec).toBe(resolvedPreset.spec);
  });

  it('records the originating preset in the tracking annotation', () => {
    const args = buildPresetCreateArgs({
      resolvedPreset,
      provider: 'psmdb',
      dbName: 'my-db',
      k8sNamespace: 'prod',
      presetName: 'psmdb-replicaset',
    });

    expect(args.annotations).toEqual({
      [INSTANCE_PRESET_ANNOTATION]: 'psmdb-replicaset',
    });
  });

  it('produces a spec of providerRef + preset spec, with no wizard meta leaking', () => {
    const args = buildPresetCreateArgs({
      resolvedPreset,
      provider: 'psmdb',
      dbName: 'my-db',
      k8sNamespace: 'prod',
      presetName: 'psmdb-replicaset',
    });

    const spec = buildCreateInstanceSpec(args.formValue);

    expect(spec).toEqual({
      ...resolvedPreset.spec,
      providerRef: { name: 'psmdb' },
    });
    // Request-address / UI-only fields must never end up in the Instance spec.
    expect(spec).not.toHaveProperty('dbName');
    expect(spec).not.toHaveProperty('k8sNamespace');
    expect(spec).not.toHaveProperty('presetName');
    expect(spec).not.toHaveProperty('provider');
  });
});
