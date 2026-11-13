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
import { Instance } from 'shared-types/api.types';
import { BackupStatus } from 'shared-types/backups.types';
import { getFixtureInstance } from 'utils/tests/instance-status-fixtures';
import { DbActions } from './db-actions';

vi.mock('hooks/api/db-instance', () => ({
  useDbInstanceActions: () => ({
    handleDeleteDbInstance: vi.fn(),
    handleRestoreDbCluster: vi.fn(),
    deleteMutation: {},
  }),
}));

vi.mock('hooks/api/backups/useBackups', () => ({
  useBackupsList: () => ({
    data: [{ status: { state: BackupStatus.SUCCEEDED } }],
  }),
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({ canDelete: true }),
}));

vi.mock('hooks/api/restores/useCanRestore', () => ({
  useCanRestore: () => true,
}));

vi.mock('hooks/api/restores/useCanCreateClusterFromBackup', () => ({
  useCanCreateClusterFromBackup: () => true,
}));

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => ({ plugins: [] }),
}));

vi.mock('./db-actions-modals', () => ({ default: () => null }));

// The actions of a database in the state a status fixture describes.
const openActions = (fixture: string) => {
  const instance: Instance = getFixtureInstance(fixture);
  render(<DbActions dbInstance={instance} />);
  fireEvent.click(screen.getByTestId('actions-menu-button'));

  return {
    restore: screen.getByTestId('db-restore'),
    createFromBackup: screen.getByTestId('db-create-new-db-from-backup'),
    remove: screen.getByTestId('db-delete'),
  };
};

describe('DbActions', () => {
  it.each([
    'healthy',
    // A database is most often restored when it is down.
    'crash-loop-all-replicas-stalled',
    'in-place-restore',
    'rolling-upgrade',
    'creating',
    'change-availability-unknown',
    'never-reconciled-fresh',
  ])('offers every action on the %s database', (fixture) => {
    const { restore, createFromBackup, remove } = openActions(fixture);

    expect(restore).not.toHaveAttribute('aria-disabled');
    expect(createFromBackup).not.toHaveAttribute('aria-disabled');
    expect(remove).not.toHaveAttribute('aria-disabled');
  });

  it.each(['deleting', 'deleting-waiting-for-children'])(
    'offers no action on the %s database',
    (fixture) => {
      const { restore, createFromBackup, remove } = openActions(fixture);

      expect(restore).toHaveAttribute('aria-disabled', 'true');
      expect(createFromBackup).toHaveAttribute('aria-disabled', 'true');
      expect(remove).toHaveAttribute('aria-disabled', 'true');
    }
  );
});
