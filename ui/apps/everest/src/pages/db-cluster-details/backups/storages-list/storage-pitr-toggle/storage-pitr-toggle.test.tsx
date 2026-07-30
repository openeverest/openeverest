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

import { render, screen, fireEvent } from '@testing-library/react';
import { Instance } from 'shared-types/api.types';
import {
  BACKUP_CLASS_NAME,
  buildBackupInstance,
  buildProviderClass,
} from 'pages/db-cluster-details/backups/__mocks__/backups-mocks';
import { StoragePitrToggle } from './storage-pitr-toggle';
import { ScheduleModalContext } from '../../backups.context';
import {
  InstanceBackupStorage,
  ScheduleModalContextType,
} from '../../backups.types';
import { Messages } from './storage-pitr-toggle.messages';
import { Messages as ToggleSwitchMessages } from 'components/pitr-toggle-switch/pitr-toggle-switch.messages';

const mockUpdateInstance = vi.fn();
const mockUseActiveBackupClass = vi.fn();
const mockUseBackupClassUiSchema = vi.fn();
const mockUseRBACPermissions = vi.fn();

vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useActiveBackupClass: () => mockUseActiveBackupClass(),
  useBackupClassUiSchema: () => mockUseBackupClassUiSchema(),
}));
vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => mockUseRBACPermissions(),
}));
vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({
    mutate: mockUpdateInstance,
    isPending: false,
  }),
}));

const withSchedule = (enabled: boolean): InstanceBackupStorage['schedules'] => [
  { name: 'daily', cron: '0 2 * * *', enabled },
];

const renderToggle = (
  storage: InstanceBackupStorage,
  instance: Instance = buildBackupInstance([storage])
) => {
  const contextValue = {
    instance,
  } as unknown as ScheduleModalContextType;

  return render(
    <ScheduleModalContext.Provider value={contextValue}>
      <StoragePitrToggle storage={storage} />
    </ScheduleModalContext.Provider>
  );
};

// The disabled switch lives inside a hover span; hovering the label surfaces
// the tooltip that explains why PITR is unavailable.
const expectTooltip = async (message: string) => {
  fireEvent.mouseOver(screen.getByText(ToggleSwitchMessages.pitr));
  expect(await screen.findByRole('tooltip')).toHaveTextContent(message);
};

beforeEach(() => {
  mockUpdateInstance.mockClear();
  mockUseActiveBackupClass.mockReturnValue(buildProviderClass());
  mockUseBackupClassUiSchema.mockReturnValue({ sections: undefined });
  mockUseRBACPermissions.mockReturnValue({ canUpdate: true });
});

describe('StoragePitrToggle', () => {
  it('renders nothing when the provider does not support PITR', () => {
    mockUseActiveBackupClass.mockReturnValue(
      buildProviderClass(BACKUP_CLASS_NAME, {
        supportsPITR: false,
        limits: { maxPITREnabledStorages: 1 },
      })
    );

    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
    });

    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
  });

  it('shows a disabled toggle when the user cannot update the instance', async () => {
    mockUseRBACPermissions.mockReturnValue({ canUpdate: false });

    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
    });

    expect(screen.getByRole('switch')).toBeDisabled();
    await expectTooltip(Messages.noPermission);
  });

  it('enables PITR by patching only the target storage', () => {
    const target: InstanceBackupStorage = {
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
    };
    const other: InstanceBackupStorage = {
      storageRef: { name: 's2' },
      schedules: withSchedule(true),
    };
    const instance = buildBackupInstance([target, other]);

    renderToggle(target, instance);
    fireEvent.click(screen.getByRole('switch'));

    expect(mockUpdateInstance).toHaveBeenCalledTimes(1);
    const patched = mockUpdateInstance.mock.calls[0][0] as Instance;
    const backup = patched.spec?.backup;
    expect(backup?.storages?.[0].pitr?.enabled).toBe(true);
    expect(backup?.storages?.[1]).toEqual(other);
    expect(backup?.classRef?.name).toBe(BACKUP_CLASS_NAME);
    expect(backup?.enabled).toBe(true);
  });

  it('is disabled and explains the missing backup schedule', async () => {
    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(false),
    });

    expect(screen.getByRole('switch')).toBeDisabled();
    await expectTooltip(Messages.needsSchedule);
  });

  it('is disabled and explains the reached PITR limit', async () => {
    const instance = buildBackupInstance([
      { storageRef: { name: 's1' }, schedules: withSchedule(true) },
      {
        storageRef: { name: 's2' },
        schedules: withSchedule(true),
        pitr: { enabled: true },
      },
    ]);

    renderToggle(instance.spec!.backup!.storages![0], instance);

    expect(screen.getByRole('switch')).toBeDisabled();
    await expectTooltip(Messages.limitReached(1));
  });

  it('prefers the schedule message when both blocks apply', async () => {
    const instance = buildBackupInstance([
      { storageRef: { name: 's1' }, schedules: withSchedule(false) },
      {
        storageRef: { name: 's2' },
        schedules: withSchedule(false),
        pitr: { enabled: true },
      },
    ]);

    renderToggle(instance.spec!.backup!.storages![0], instance);

    await expectTooltip(Messages.needsSchedule);
  });

  it('confirms before disabling, then patches enabled: false', () => {
    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
      pitr: { enabled: true },
    });

    // Turning off does not patch immediately — it asks for confirmation first.
    fireEvent.click(screen.getByRole('switch'));
    expect(mockUpdateInstance).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('confirm-dialog-disable'));

    expect(mockUpdateInstance).toHaveBeenCalledTimes(1);
    const patched = mockUpdateInstance.mock.calls[0][0] as Instance;
    expect(patched.spec?.backup?.storages?.[0].pitr?.enabled).toBe(false);
  });

  it('shows a config button that opens the modal when enabled with a schema', () => {
    mockUseBackupClassUiSchema.mockReturnValue({
      sections: { pitr: { label: 'PITR', components: {} } },
    });

    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
      pitr: { enabled: true },
    });

    fireEvent.click(screen.getByTestId('pitr-configure-s1'));

    expect(screen.getByText(/Configure PITR/)).toBeInTheDocument();
  });

  it('keeps the config button present but disabled when PITR is off', () => {
    mockUseBackupClassUiSchema.mockReturnValue({
      sections: { pitr: { label: 'PITR', components: {} } },
    });

    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
    });

    expect(screen.getByTestId('pitr-configure-s1')).toBeDisabled();
  });

  it('keeps the config button disabled with a tooltip when the user cannot update', async () => {
    mockUseRBACPermissions.mockReturnValue({ canUpdate: false });
    mockUseBackupClassUiSchema.mockReturnValue({
      sections: { pitr: { label: 'PITR', components: {} } },
    });

    renderToggle({
      storageRef: { name: 's1' },
      schedules: withSchedule(true),
      pitr: { enabled: true },
    });

    const configButton = screen.getByTestId('pitr-configure-s1');
    expect(configButton).toBeDisabled();

    fireEvent.mouseOver(configButton.parentElement!);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      Messages.noPermission
    );
  });
});
