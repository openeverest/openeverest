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

import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { Backup, BackupSourceMetadata } from 'shared-types/backups.types';
import { ImportSource } from 'pages/database-form/import-navigation-state';
import { ImportPrefill } from 'pages/database-form/utils/import-prefill';
import { ImportSourceBanner } from './import-source-banner';
import { Messages } from './import-source-banner.messages';

const IMPORT_SOURCE: ImportSource = {
  namespace: 'ns-a',
  backupClassName: 'psmdb-pbm',
  dataSource: {
    type: 'Backup',
    backupName: 'import-abc12-1',
    storageName: 's3-old',
  },
};
const ROUTER_STATE = { importSource: IMPORT_SOURCE };
const PATH = 'orders-db/2026-10-17T02:00:04Z';
const BACKUP: Backup = {
  metadata: { name: 'import-abc12-1' },
  spec: {
    classRef: { name: 'psmdb-pbm' },
    storageRef: { name: 's3-old' },
    origin: {
      type: 'External',
      external: {
        path: PATH,
        startedAt: '2026-10-17T02:00:00Z',
        completedAt: '2026-10-17T02:00:04Z',
      },
    },
  },
};

const mocks = vi.hoisted(() => ({ navigate: vi.fn() }));

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => mocks.navigate,
  useLocation: () => ({ state: ROUTER_STATE }),
}));
vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));
vi.mock('hooks/api/backups/useBackups', () => ({
  useGetBackup: () => ({ data: BACKUP }),
}));

describe('ImportSourceBanner', () => {
  const renderBanner = (
    sourceMetadata?: BackupSourceMetadata,
    prefill: ImportPrefill = {}
  ) =>
    render(
      <TestWrapper>
        <ImportSourceBanner
          importSource={{ ...IMPORT_SOURCE, sourceMetadata }}
          prefill={prefill}
        />
      </TestWrapper>
    );

  it('names the backup by the path it was picked by and goes back to change it', () => {
    renderBanner();

    expect(
      screen.getByText(Messages.source(PATH, 's3-old'), { exact: false })
    ).toBeInTheDocument();

    fireEvent.click(screen.getByText(Messages.changeBackup));
    expect(mocks.navigate).toHaveBeenCalledWith('/databases/import', {
      state: ROUTER_STATE,
    });
  });

  it.each<[string, BackupSourceMetadata | undefined, ImportPrefill, string[]]>([
    [
      'a matching version',
      { engineVersion: '8.0.12-4', topology: 'replicaSet' },
      { version: '8.0.12', topology: 'replicaSet' },
      [Messages.versionMatched('8.0.12')],
    ],
    [
      'a version that is not available',
      { engineVersion: '8.0.19-7' },
      {},
      [Messages.versionMissing('8.0.19-7'), Messages.topologyUnknown],
    ],
    [
      'no recorded version',
      undefined,
      {},
      [Messages.versionUnknown, Messages.topologyUnknown],
    ],
  ])('says what to pick for %s', (_, sourceMetadata, prefill, hints) => {
    renderBanner(sourceMetadata, prefill);

    expect(
      screen.getByText([Messages.source(PATH, 's3-old'), ...hints].join(' '))
    ).toBeInTheDocument();
  });
});
