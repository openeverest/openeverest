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

import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { PitrBlock } from './pitr-block';
import { Messages } from './pitr-block.messages';

const mockUseBackupClassesList = vi.fn();
const mockUseBackupClassUiSchema = vi.fn();

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));
vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useBackupClassesList: () => mockUseBackupClassesList(),
  useBackupClassUiSchema: () => mockUseBackupClassUiSchema(),
}));
// The config modal is exercised on its own; here it would only pull in the
// form-dialog/UIGenerator machinery, so stub it out.
vi.mock('components/pitr-config-modal', () => ({
  PitrConfigModal: () => null,
}));

const providerClass = (providerManaged: Record<string, unknown> = {}) => ({
  metadata: { name: 'cls' },
  spec: {
    providerManaged: {
      supportsPITR: true,
      limits: { maxPITREnabledStorages: 1 },
      ...providerManaged,
    },
  },
});

const scheduleFor = (storageName: string) => ({
  name: `sch-${storageName}`,
  cron: '0 2 * * *',
  enabled: true,
  storageName,
});

interface RenderOptions {
  schedules?: ReturnType<typeof scheduleFor>[];
  pitr?: Record<string, { enabled: boolean }>;
  providerManaged?: Record<string, unknown>;
  sections?: Record<string, unknown>;
}

const PitrKeysProbe = () => {
  const map = useWatch({ name: 'backup.pitr' }) as
    | Record<string, { enabled?: boolean } | undefined>
    | undefined;
  const enabled = Object.entries(map ?? {})
    .filter(([, config]) => config?.enabled)
    .map(([name]) => name)
    .join(',');
  return <div data-testid="pitr-enabled-keys">{enabled}</div>;
};

const renderBlock = ({
  schedules = [],
  pitr = {},
  providerManaged = {},
  sections,
}: RenderOptions = {}) => {
  mockUseBackupClassesList.mockReturnValue({
    data: [providerClass(providerManaged)],
  });
  mockUseBackupClassUiSchema.mockReturnValue({ sections });

  const Harness = () => {
    const methods = useForm({
      defaultValues: {
        k8sNamespace: 'ns',
        backup: { classRef: { name: 'cls' }, schedules, pitr },
      },
    });
    return (
      <FormProvider {...methods}>
        <button onClick={() => methods.setValue('backup.schedules', [])}>
          clear-schedules
        </button>
        <PitrKeysProbe />
        <PitrBlock />
      </FormProvider>
    );
  };

  return render(<Harness />);
};

const switchOf = (storageName: string) =>
  within(screen.getByTestId(`pitr-toggle-${storageName}`)).getByRole('switch');

beforeEach(() => {
  mockUseBackupClassesList.mockReset();
  mockUseBackupClassUiSchema.mockReset();
});

describe('PitrBlock', () => {
  it('renders nothing when the provider does not support PITR', () => {
    renderBlock({
      schedules: [scheduleFor('s3')],
      providerManaged: { supportsPITR: false },
    });

    expect(screen.queryByText(Messages.title)).not.toBeInTheDocument();
  });

  it('renders nothing when there are no active schedules', () => {
    renderBlock({ schedules: [] });

    expect(screen.queryByText(Messages.title)).not.toBeInTheDocument();
  });

  it('renders a toggle and name for each scheduled storage', () => {
    renderBlock({ schedules: [scheduleFor('s1'), scheduleFor('s2')] });

    expect(screen.getByTestId('pitr-toggle-s1')).toBeInTheDocument();
    expect(screen.getByTestId('pitr-toggle-s2')).toBeInTheDocument();
    expect(screen.getByText('Storage: s1')).toBeInTheDocument();
  });

  it('hides the config gear when the class ships no PITR schema', () => {
    renderBlock({ schedules: [scheduleFor('s3')], sections: undefined });

    expect(
      screen.queryByTestId('edit-editable-item-button-s3')
    ).not.toBeInTheDocument();
  });

  it('shows a disabled config gear when PITR is off but a schema exists', () => {
    renderBlock({
      schedules: [scheduleFor('s3')],
      sections: { pitr: { label: 'PITR', components: {} } },
    });

    expect(screen.getByTestId('edit-editable-item-button-s3')).toBeDisabled();
  });

  it('disables enabling more storages than the provider limit allows', () => {
    renderBlock({
      schedules: [scheduleFor('s1'), scheduleFor('s2')],
      pitr: { s1: { enabled: true } },
      providerManaged: { limits: { maxPITREnabledStorages: 1 } },
    });

    expect(switchOf('s1')).not.toBeDisabled();
    expect(switchOf('s2')).toBeDisabled();
  });

  it('writes the enabled storage into backup.pitr on toggle', () => {
    renderBlock({ schedules: [scheduleFor('s3')] });

    fireEvent.click(switchOf('s3'));

    expect(screen.getByTestId('pitr-enabled-keys')).toHaveTextContent('s3');
  });

  it('clears a storage PITR entry when its schedule is removed', async () => {
    renderBlock({
      schedules: [scheduleFor('s3')],
      pitr: { s3: { enabled: true } },
    });
    expect(screen.getByTestId('pitr-enabled-keys')).toHaveTextContent('s3');

    fireEvent.click(screen.getByText('clear-schedules'));

    await waitFor(() =>
      expect(screen.getByTestId('pitr-enabled-keys')).not.toHaveTextContent(
        's3'
      )
    );
  });
});
