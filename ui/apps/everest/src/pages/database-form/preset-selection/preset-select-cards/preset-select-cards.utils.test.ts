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

import { InstancePreset } from 'shared-types/api.types';
import { presetToCardItem } from './preset-select-cards.utils';

const preset: InstancePreset = {
  metadata: { name: 'psmdb-prod' },
  spec: {
    providerRef: { name: 'percona-server-mongodb' },
    version: '8.0.12',
    topology: { type: 'replicaSet' },
    components: {
      engine: {
        replicas: 3,
        resources: { limits: { cpu: 1, memory: '4Gi' } },
        storage: { size: '25Gi' },
      },
    },
  },
};

describe('presetToCardItem', () => {
  it('maps the id/title and the displayed meta + resources', () => {
    const item = presetToCardItem(preset);

    expect(item.id).toBe('psmdb-prod');
    expect(item.title).toBe('psmdb-prod');
    expect(item.subtitle).toBe('replicaSet · 8.0.12');
    expect(item.detail).toBe('3 nodes · 1 CPU · 4Gi RAM · 25Gi disk');
  });

  it('builds searchText from the whole spec, including fields not shown', () => {
    const haystack = presetToCardItem(preset).searchText?.toLowerCase() ?? '';

    // providerRef.name is searchable but is never rendered on the card.
    expect(haystack).toContain('percona-server-mongodb');
    expect(haystack).toContain('providerref');
    expect(haystack).toContain('storage');
  });
});
