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

import { Provider } from 'shared-types/api.types';
import { BackupSourceMetadata } from 'shared-types/backups.types';
import { getImportPrefill } from './import-prefill';

const PROVIDER: Provider = {
  metadata: { name: 'psmdb' },
  spec: {
    versions: [
      {
        name: '8.0.12',
        components: { engine: '8.0.12-4', backupAgent: '2.12.0' },
      },
      {
        name: '7.0.18',
        components: { engine: '7.0.18-11', backupAgent: '2.12.0' },
      },
    ],
  },
};
const TOPOLOGIES = ['replicaSet', 'sharded'];

describe('getImportPrefill', () => {
  it('takes the topology and the version bundle the backup was taken on', () => {
    expect(
      getImportPrefill(
        {
          engineVersion: '7.0.18-11',
          backupType: 'logical',
          topology: 'sharded',
        },
        PROVIDER,
        TOPOLOGIES
      )
    ).toEqual({ topology: 'sharded', version: '7.0.18' });
  });

  it.each<[string, BackupSourceMetadata | undefined, object]>([
    [
      'a version no bundle has',
      { engineVersion: '8.0.19-7', topology: 'replicaSet' },
      { topology: 'replicaSet', version: undefined },
    ],
    ['only a major version', { engineVersion: '8' }, { version: undefined }],
    [
      "another component's version",
      { engineVersion: '2.12.0' },
      { version: undefined },
    ],
    [
      'a topology the provider lacks',
      { engineVersion: '8.0.12-4', topology: 'standalone' },
      { topology: undefined, version: '8.0.12' },
    ],
    ['no metadata', undefined, { topology: undefined, version: undefined }],
  ])('leaves out what it cannot match: %s', (_, sourceMetadata, expected) => {
    expect(
      getImportPrefill(sourceMetadata, PROVIDER, TOPOLOGIES)
    ).toMatchObject(expected);
  });
});
