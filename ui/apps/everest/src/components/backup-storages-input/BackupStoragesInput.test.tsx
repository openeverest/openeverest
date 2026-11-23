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
  act,
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm, useFormState, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { storageLocationZodObject } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Messages as StatusMessages } from 'utils/backup-storages.messages';
import { Messages } from './backup-storages-input.messages';
import BackupStoragesInput from '.';

const queryClient = new QueryClient();

// The namespace's storages, and those each backup class can use.
const mocks = vi.hoisted(() => {
  const condition = (
    type: string,
    status: 'True' | 'False',
    reason: string
  ) => ({
    type,
    status,
    reason,
    message: '',
    lastTransitionTime: '',
  });
  const storage = (
    name: string,
    accepted = condition('Accepted', 'True', 'Accepted'),
    deletionTimestamp?: string
  ): BackupStorageCRD => ({
    metadata: { name, deletionTimestamp },
    spec: {
      backupStorageClassRef: { name: 's3' },
      authentication: { mode: 'AccessKey' },
      location: { objectStore: { bucket: name } },
    },
    status: { conditions: [accepted] },
  });
  const list = (data: BackupStorageCRD[]) => ({
    data,
    isFetching: false,
    isSuccess: true,
  });
  const lists: Record<string, ReturnType<typeof list>> = {
    namespace: list(
      ['storage1', 'storage2', 'storage3', 'storage4'].map((name) =>
        storage(name)
      )
    ),
    'pg-class': list([storage('s3-pg')]),
    'empty-class': list([]),
    'broken-class': list([
      storage(
        's3-broken',
        condition('Accepted', 'False', 'CredentialsNotFound')
      ),
    ]),
    'mongo-class': list([
      storage(
        's3-broken',
        condition('Accepted', 'False', 'CredentialsNotFound')
      ),
      storage('s3-mongo'),
      storage('s3-going', undefined, '2026-11-16T10:00:00Z'),
    ]),
  };
  return { lists, storage };
});

vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
  useBackupStoragesByNamespace: (
    _namespace: string,
    filter?: { backupClass?: string }
  ) => mocks.lists[filter?.backupClass ?? 'namespace'],
}));

const FormProviderWrapper = ({
  children,
  storageLocation = null,
}: {
  children: React.ReactNode;
  storageLocation?: BackupStorageCRD | null;
}) => {
  const methods = useForm({
    defaultValues: {
      storageLocation,
    },
  });
  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

const openOptions = () =>
  fireEvent.click(
    screen.getAllByRole('button').find((el) => el.title === 'Open')!
  );

// --- Unit tests for getAvailableStorages utility ---

// --- Component integration tests ---

describe('BackupStoragesInput (on-demand context)', () => {
  it(
    'shows all storages and helper text when limit > active',
    { timeout: 30000 },
    () => {
      render(
        <FormProviderWrapper>
          <QueryClientProvider client={queryClient}>
            <BackupStoragesInput
              namespace="test"
              schedules={[]}
              maxStorages={2}
              instanceStorageNames={['storage1']}
            />
          </QueryClientProvider>
        </FormProviderWrapper>
      );

      expect(
        screen.getByText(
          'You are currently using 1 out of 2 available storages.'
        )
      ).toBeInTheDocument();

      fireEvent.click(
        screen.getAllByRole('button').find((el) => el.title === 'Open')!
      );
      expect(screen.getAllByRole('option').length).toBe(4);
    }
  );

  it('leaves the field empty once the user clears it', async () => {
    const picker = () => (
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput namespace="test" schedules={[]} />
        </QueryClientProvider>
      </FormProviderWrapper>
    );
    const { rerender } = render(picker());
    const field = screen.getByRole('combobox');
    await waitFor(() => expect(field).toHaveValue('storage1'));

    fireEvent.click(screen.getByTitle('Clear'));
    // The form around the picker renders again on any change of its values.
    rerender(picker());

    await waitFor(() => expect(field).toHaveValue(''));
    await act(() => new Promise((resolve) => setTimeout(resolve, 50)));
    expect(field).toHaveValue('');
  });

  it('shows "(in use)" highlight for instance storages when limit not reached', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={3}
            instanceStorageNames={['storage1', 'storage2']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    // All 4 storages shown
    expect(screen.getAllByRole('option').length).toBe(4);
    // In-use labels shown for storage1 and storage2
    expect(screen.getAllByText('(in use)').length).toBe(2);
  });

  it('shows only instance storages when limit == active > 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={2}
            instanceStorageNames={['storage1', 'storage3']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText(
        'Storage limit reached. You are using 2 out of 2 available storages.'
      )
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(2);
    // No "(in use)" labels when showing restricted list
    expect(screen.queryByText('(in use)')).not.toBeInTheDocument();
  });

  it('disables field and prefills when limit == active == 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={1}
            instanceStorageNames={['storage2']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText(
        'Storage limit reached. You are using 1 out of 1 available storages.'
      )
    ).toBeInTheDocument();

    // Field should be disabled
    const input = screen.getByRole('combobox');
    expect(input).toBeDisabled();
    expect(input).toHaveValue('storage2');
  });

  it('shows all storages when active == 0 and limit == 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={1}
            instanceStorageNames={[]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText('You are currently using 0 out of 1 available storages.')
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(4);
  });

  it('shows all storages with no helper text when maxStorages is undefined', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            instanceStorageNames={['storage1']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(screen.queryByText(/available storages/)).not.toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(4);
  });
});

describe('BackupStoragesInput (backup class)', () => {
  const renderForClass = (backupClass: string) =>
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            backupClass={backupClass}
            schedules={[]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

  it('offers only the storages the backup class can use', () => {
    renderForClass('pg-class');

    openOptions();
    expect(
      screen.getAllByRole('option').map((option) => option.textContent)
    ).toEqual(['s3-pg']);
  });

  it("offers a storage that isn't valid as unavailable, saying why, and picks a valid one", () => {
    renderForClass('mongo-class');

    expect(screen.getByRole('combobox')).toHaveValue('s3-mongo');
    openOptions();
    const broken = screen.getByRole('option', { name: /s3-broken/ });
    expect(broken).toHaveAttribute('aria-disabled', 'true');
    expect(
      within(broken).getByText(StatusMessages.reasons.CredentialsNotFound)
    ).toBeInTheDocument();
  });

  it('offers a storage being deleted only to an instance that uses it', () => {
    const { unmount } = renderForClass('mongo-class');
    openOptions();
    expect(
      screen.queryByRole('option', { name: /s3-going/ })
    ).not.toBeInTheDocument();
    unmount();

    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            backupClass="mongo-class"
            schedules={[]}
            instanceStorageNames={['s3-going']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );
    openOptions();
    expect(screen.getByRole('option', { name: /s3-going/ })).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it("swaps a picked storage the new backup class can't use", () => {
    const picker = (backupClass: string) => (
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            backupClass={backupClass}
            schedules={[]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );
    const { rerender } = render(picker('pg-class'));
    expect(screen.getByRole('combobox')).toHaveValue('s3-pg');

    rerender(picker('mongo-class'));

    expect(screen.getByRole('combobox')).toHaveValue('s3-mongo');
  });

  it('keeps the storage of an existing schedule the class list lacks', () => {
    render(
      <FormProviderWrapper storageLocation={mocks.storage('s3-pg')}>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            backupClass="mongo-class"
            schedules={[]}
            autoFillProps={{ disabled: true }}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(screen.getByRole('combobox')).toHaveValue('s3-pg');
  });

  const ClassAndStorageForm = ({ initialClass }: { initialClass: string }) => {
    const methods = useForm({
      mode: 'onChange',
      resolver: zodResolver(
        z.object({
          backupClass: z.string(),
          // A union, as in the schedule and on-demand forms, validates the
          // old pick slowly enough to settle after the field is cleared.
          storageLocation: storageLocationZodObject.refine(
            (storage) => !!storage,
            'Select a storage'
          ),
        })
      ),
      defaultValues: { backupClass: initialClass, storageLocation: null },
    });
    const backupClass = useWatch({
      control: methods.control,
      name: 'backupClass',
    });
    return (
      <FormProvider {...methods}>
        <select data-testid="class" {...methods.register('backupClass')}>
          <option value="pg-class">pg-class</option>
          <option value="empty-class">empty-class</option>
          <option value="broken-class">broken-class</option>
        </select>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            backupClass={backupClass}
            schedules={[]}
          />
        </QueryClientProvider>
        <FormValidity />
      </FormProvider>
    );
  };
  const FormValidity = () => {
    const { isValid } = useFormState();
    return <span data-testid="form-valid">{String(isValid)}</span>;
  };
  const storageField = () =>
    screen.getByRole('combobox', { name: /backup storage/i });
  const pickClass = (backupClass: string) =>
    fireEvent.change(screen.getByTestId('class'), {
      target: { value: backupClass },
    });
  const expectFormValid = (isValid: boolean) =>
    waitFor(() =>
      expect(screen.getByTestId('form-valid')).toHaveTextContent(
        String(isValid)
      )
    );

  // Lets every validation a class change started settle, so a test sees
  // where the form ends up rather than a state it passes through.
  const settle = () =>
    act(() => new Promise<void>((resolve) => setTimeout(resolve)));
  const expectSettledForm = (
    isValid: boolean,
    storage: string,
    explained: boolean
  ) => {
    expect(screen.getByTestId('form-valid')).toHaveTextContent(String(isValid));
    expect(storageField()).toHaveValue(storage);
    const explanation = screen.queryByText(Messages.noCompatibleStorages);
    if (explained) {
      expect(explanation).toBeInTheDocument();
    } else {
      expect(explanation).not.toBeInTheDocument();
    }
  };

  it('keeps a form invalid once the picked backup class can use no storage, saying why', async () => {
    render(<ClassAndStorageForm initialClass="pg-class" />);
    await expectFormValid(true);

    pickClass('empty-class');
    await settle();

    expectSettledForm(false, '', true);
  });

  it('keeps a form invalid once the picked backup class offers only unavailable storages', async () => {
    render(<ClassAndStorageForm initialClass="pg-class" />);
    await expectFormValid(true);

    pickClass('broken-class');
    await settle();

    expectSettledForm(false, '', false);
  });

  it('makes a form valid again once a backup class with storages is picked back', async () => {
    render(<ClassAndStorageForm initialClass="pg-class" />);
    await expectFormValid(true);
    pickClass('empty-class');
    await settle();

    pickClass('pg-class');
    await settle();

    expectSettledForm(true, 's3-pg', false);
  });

  it("doesn't explain an empty field once a backup class with storages was picked meanwhile", async () => {
    render(<ClassAndStorageForm initialClass="pg-class" />);
    await expectFormValid(true);

    pickClass('empty-class');
    pickClass('pg-class');
    await settle();

    expectSettledForm(true, 's3-pg', false);
  });
});
