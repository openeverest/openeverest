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

import { buildRestoreDataSource } from 'hooks/api/restores/restore-data-source';
import { Provider } from 'shared-types/api.types';
import {
  buildImportNavigationRouteState,
  ImportSource,
  parseImportNavigationRouteState,
} from './import-navigation-state';

const importSource: ImportSource = {
  namespace: 'ns-a',
  backupClassName: 'psmdb-pbm',
  dataSource: {
    type: 'Backup',
    backupName: 'import-x7k2p-1',
    storageName: 's3-old-bucket',
  },
};
const provider: Provider = {
  metadata: { name: 'percona-server-mongodb' },
  spec: {},
};

describe('import-navigation-state', () => {
  it('round-trips the import source through router state', () => {
    const state = buildImportNavigationRouteState(provider, importSource);

    expect(state.selectedDbProvider).toBe(provider);
    expect(parseImportNavigationRouteState(state)).toEqual({
      providerName: 'percona-server-mongodb',
      importSource,
    });
  });

  it('carries a data source the restore payload builder accepts', () => {
    const parsed = parseImportNavigationRouteState(
      buildImportNavigationRouteState(provider, importSource)
    );

    expect(
      parsed && buildRestoreDataSource(parsed.importSource.dataSource)
    ).toEqual({
      type: 'Backup',
      backup: { backupRef: { name: 'import-x7k2p-1' } },
    });
  });

  it.each([
    ['no state', undefined],
    ['a different flow', { selectedDbCluster: 'db', namespace: 'ns-a' }],
    ['a source without its provider', { importSource }],
    [
      'an empty field',
      {
        selectedDbProvider: provider,
        importSource: { ...importSource, namespace: '' },
      },
    ],
    [
      'a source without its storage',
      {
        selectedDbProvider: provider,
        importSource: {
          ...importSource,
          dataSource: { type: 'Backup', backupName: 'import-x7k2p-1' },
        },
      },
    ],
    [
      'a point-in-time source',
      {
        selectedDbProvider: provider,
        importSource: {
          ...importSource,
          dataSource: { ...importSource.dataSource, type: 'PointInTime' },
        },
      },
    ],
  ])('rejects %s', (_, state) => {
    expect(parseImportNavigationRouteState(state)).toBeUndefined();
  });
});
