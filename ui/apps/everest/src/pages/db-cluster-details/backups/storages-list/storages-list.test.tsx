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

import { render, screen } from '@testing-library/react';
import { Instance } from 'shared-types/api.types';
import { BackupStatus } from 'shared-types/backups.types';
import { WizardMode } from 'shared-types/wizard.types';
import {
  BACKUP_CLASS_NAME,
  buildBackupInstance,
  buildProviderClass,
} from 'pages/db-cluster-details/backups/__mocks__/backups-mocks';
import { StoragesList } from './storages-list';
import { ScheduleModalContext } from '../backups.context';
import { ScheduleModalContextType } from '../backups.types';
import { Messages } from './storages-list.messages';

// The PITR toggle is data-driven (own hooks); it's covered by its own tests.
// Stub it here so the list rendering is tested in isolation.
vi.mock('./storage-pitr-toggle', () => ({
  StoragePitrToggle: ({
    storage,
  }: {
    storage: { storageRef: { name: string } };
  }) => <div data-testid={`storage-row-${storage.storageRef.name}`} />,
}));

const mockUseActiveBackupClass = vi.fn();
const mockUseBackupsList = vi.fn();

vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useActiveBackupClass: () => mockUseActiveBackupClass(),
}));
vi.mock('hooks/api/backups/useBackups', () => ({
  useBackupsList: () => mockUseBackupsList(),
}));
vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const scheduledStorage = (name: string) => ({
  storageRef: { name },
  schedules: [{ name: 'daily', cron: '0 2 * * *', enabled: true }],
});

const renderWithContext = (instance: Instance) => {
  const contextValue: ScheduleModalContextType = {
    instance,
    mode: WizardMode.New,
    setMode: vi.fn(),
    selectedScheduleName: '',
    setSelectedScheduleName: vi.fn(),
    openScheduleModal: false,
    setOpenScheduleModal: vi.fn(),
    openOnDemandModal: false,
    setOpenOnDemandModal: vi.fn(),
  };

  return render(
    <ScheduleModalContext.Provider value={contextValue}>
      <StoragesList />
    </ScheduleModalContext.Provider>
  );
};

beforeEach(() => {
  // Default: provider without PITR support and no backups — no warning.
  mockUseActiveBackupClass.mockReturnValue(undefined);
  mockUseBackupsList.mockReturnValue({ data: [] });
});

describe('StoragesList', () => {
  it('renders a row per instance storage', () => {
    renderWithContext(
      buildBackupInstance([
        { storageRef: { name: 's3-prod' } },
        { storageRef: { name: 's3-archive' } },
      ])
    );

    expect(screen.getByTestId('storage-row-s3-prod')).toBeInTheDocument();
    expect(screen.getByTestId('storage-row-s3-archive')).toBeInTheDocument();
  });

  it('renders nothing when the instance has no storages', () => {
    const { container } = renderWithContext(buildBackupInstance([]));

    expect(container).toBeEmptyDOMElement();
  });

  it('warns when PITR is supported but there is no successful backup', () => {
    mockUseActiveBackupClass.mockReturnValue(buildProviderClass());

    renderWithContext(
      buildBackupInstance([{ storageRef: { name: 's3-prod' } }])
    );

    expect(screen.getByTestId('pitr-needs-backup-warning')).toHaveTextContent(
      Messages.needsBackup
    );
  });

  it('does not warn once a successful backup exists', () => {
    mockUseActiveBackupClass.mockReturnValue(buildProviderClass());
    mockUseBackupsList.mockReturnValue({
      data: [{ status: { state: BackupStatus.SUCCEEDED } }],
    });

    renderWithContext(
      buildBackupInstance([{ storageRef: { name: 's3-prod' } }])
    );

    expect(
      screen.queryByTestId('pitr-needs-backup-warning')
    ).not.toBeInTheDocument();
  });

  it('still warns when the only backup failed', () => {
    mockUseActiveBackupClass.mockReturnValue(buildProviderClass());
    mockUseBackupsList.mockReturnValue({
      data: [{ status: { state: BackupStatus.FAILED } }],
    });

    renderWithContext(
      buildBackupInstance([{ storageRef: { name: 's3-prod' } }])
    );

    expect(screen.getByTestId('pitr-needs-backup-warning')).toBeInTheDocument();
  });

  it('does not warn when the provider does not support PITR', () => {
    mockUseActiveBackupClass.mockReturnValue(
      buildProviderClass(BACKUP_CLASS_NAME, {
        supportsPITR: false,
        limits: { maxPITREnabledStorages: 1 },
      })
    );

    renderWithContext(
      buildBackupInstance([{ storageRef: { name: 's3-prod' } }])
    );

    expect(
      screen.queryByTestId('pitr-needs-backup-warning')
    ).not.toBeInTheDocument();
  });

  it('still warns with a schedule but no successful backup', () => {
    mockUseActiveBackupClass.mockReturnValue(buildProviderClass());

    renderWithContext(buildBackupInstance([scheduledStorage('s3-prod')]));

    expect(screen.getByTestId('pitr-needs-backup-warning')).toBeInTheDocument();
  });
});
