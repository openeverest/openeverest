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
import { InstanceCondition } from 'shared-types/instance.types';
import { WizardMode } from 'shared-types/wizard.types';
import { buildBackupInstance } from 'pages/db-cluster-details/backups/__mocks__/backups-mocks';
import { ScheduleModalContext } from '../../backups.context';
import BackupListTableHeader from './backups-list-table-header';
import { Messages } from './backups-list-table-header.messages';

vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useActiveBackupClass: () => undefined,
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({ canCreate: true }),
}));

const condition = (
  type: string,
  status: InstanceCondition['status'],
  reason = type
): InstanceCondition => ({
  type,
  status,
  reason,
  message: '',
  lastTransitionTime: '2026-03-01T11:00:00Z',
});

const instanceWith = (
  conditions: InstanceCondition[],
  metadata: Instance['metadata'] = {}
): Instance => ({
  ...buildBackupInstance([]),
  metadata: { name: 'db', namespace: 'demo', ...metadata },
  status: { conditions },
});

const onNowClick = vi.fn();

const renderHeader = (instance: Instance) =>
  render(
    <ScheduleModalContext.Provider
      value={{
        instance,
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
      <BackupListTableHeader
        onNowClick={onNowClick}
        onScheduleClick={vi.fn()}
      />
    </ScheduleModalContext.Provider>
  );

const openMenu = () => fireEvent.click(screen.getByTestId('menu-button'));

describe('BackupListTableHeader', () => {
  beforeEach(() => {
    onNowClick.mockClear();
  });

  it.each([
    ['is available', [condition('Available', 'True')]],
    // A provider that can't tell leaves the decision to the engine.
    ['may be available', [condition('Available', 'Unknown', 'NotReported')]],
    ['does not report availability', []],
    // Converging is no reason to hold a backup back.
    [
      'is being upgraded',
      [
        condition('Available', 'True'),
        condition('Ready', 'False', 'Upgrading'),
        condition('Reconciling', 'True', 'Upgrading'),
      ],
    ],
  ])('takes a backup now of a database that %s', (_, conditions) => {
    renderHeader(instanceWith(conditions));
    openMenu();

    const now = screen.getByTestId('now-menu-item');
    expect(now).not.toHaveAttribute('aria-disabled');
    fireEvent.click(now);

    expect(onNowClick).toHaveBeenCalled();
  });

  it('cannot take a backup now of a database that is down, and says why', () => {
    renderHeader(
      instanceWith([condition('Available', 'False', 'NotAvailable')])
    );
    openMenu();

    const now = screen.getByTestId('now-menu-item');
    expect(now).toHaveAttribute('aria-disabled', 'true');
    expect(now).toHaveTextContent(Messages.notAvailable);
  });

  it('still schedules backups of a database that is down', () => {
    renderHeader(
      instanceWith([condition('Available', 'False', 'NotAvailable')])
    );
    openMenu();

    expect(screen.getByTestId('schedule-menu-item')).not.toHaveAttribute(
      'aria-disabled'
    );
  });

  it('creates no backup for a database that is being deleted', () => {
    renderHeader(
      instanceWith([condition('Available', 'True')], {
        deletionTimestamp: '2026-03-01T11:59:00Z',
      })
    );

    expect(screen.getByTestId('menu-button')).toBeDisabled();
  });
});
