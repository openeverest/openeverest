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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { WizardMode } from 'shared-types/wizard.types';
import { TestWrapper } from 'utils/test';
import { getFixtureInstance } from 'utils/tests/instance-status-fixtures';
import { ScheduleModalContext } from '../backups.context';
import { BackupsList } from './backups-list';

vi.mock('hooks/api/backups/useBackups', () => ({
  getBackupListQueryKey: () => ['backups'],
  useBackupsList: () => ({ data: [] }),
  useDeleteBackup: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({ mutate: vi.fn() }),
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({ canCreate: true, canDelete: true }),
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

vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useActiveBackupClass: () => undefined,
}));

// The backups of a database in the state a status fixture describes.
const renderBackups = (fixture: string) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TestWrapper>
        <ScheduleModalContext.Provider
          value={{
            instance: getFixtureInstance(fixture),
            mode: WizardMode.New,
            setMode: vi.fn(),
            selectedScheduleName: '',
            setSelectedScheduleName: vi.fn(),
            openScheduleModal: false,
            setOpenScheduleModal: vi.fn(),
            openOnDemandModal: false,
            setOpenOnDemandModal: vi.fn(),
          }}
        >
          <BackupsList />
        </ScheduleModalContext.Provider>
      </TestWrapper>
    </QueryClientProvider>
  );

describe('BackupsList', () => {
  it.each([
    ['in-place-restore', 'Restore nightly-0412 is running'],
    // Loading a new database from a backup is a restore too.
    ['seeding-from-backup', 'Restoring from backup nightly-0412'],
  ])('says that a restore is in progress (%s)', (fixture, message) => {
    renderBackups(fixture);

    expect(screen.getByTestId('restore-in-progress')).toHaveTextContent(
      message
    );
  });

  it.each(['healthy', 'rolling-upgrade', 'crash-loop-all-replicas-stalled'])(
    'reports no restore for the %s database',
    (fixture) => {
      renderBackups(fixture);

      expect(
        screen.queryByTestId('restore-in-progress')
      ).not.toBeInTheDocument();
    }
  );
});
