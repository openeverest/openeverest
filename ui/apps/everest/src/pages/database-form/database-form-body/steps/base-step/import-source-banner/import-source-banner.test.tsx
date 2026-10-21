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
import { Backup } from 'shared-types/backups.types';
import { ImportSource } from 'pages/database-form/import-navigation-state';
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
const BACKUP: Backup = {
  metadata: { name: 'import-abc12-1' },
  spec: {
    classRef: { name: 'psmdb-pbm' },
    storageRef: { name: 's3-old' },
    origin: {
      type: 'External',
      external: {
        path: 'orders-db/2026-10-17T02:00:04Z',
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
  it('names the backup by the path it was picked by and goes back to change it', () => {
    render(
      <TestWrapper>
        <ImportSourceBanner importSource={IMPORT_SOURCE} />
      </TestWrapper>
    );

    expect(
      screen.getByText(
        Messages.source('orders-db/2026-10-17T02:00:04Z', 's3-old')
      )
    ).toBeInTheDocument();

    fireEvent.click(screen.getByText(Messages.changeBackup));
    expect(mocks.navigate).toHaveBeenCalledWith('/databases/import', {
      state: ROUTER_STATE,
    });
  });
});
