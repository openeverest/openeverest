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

// @ts-nocheck
// TODO remove this file after release of v2
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { DbType } from '@percona/types';
import { TimeValue } from 'components/time-selection/time-selection.types';
import { FormProvider, useForm } from 'react-hook-form';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TestWrapper } from 'utils/test';
import { Backups } from './backups.js';
import { StorageType } from 'shared-types/backupStorages.types.ts';
import { PG_SLOTS_LIMIT } from 'consts.ts';
import { DbWizardType } from 'pages/database-form/database-form-schema.ts';

const backupStoragesMocks = vi.hoisted(() => ({
  useBackupStoragesByNamespace: vi.fn().mockReturnValue({}),
}));

vi.mock('hooks/api/backup-storages/useBackupStorages', async () => {
  const actual = await vi.importActual(
    'hooks/api/backup-storages/useBackupStorages'
  );
  return {
    ...actual,
    useBackupStoragesByNamespace:
      backupStoragesMocks.useBackupStoragesByNamespace,
  };
});
vi.mock('hooks/api/db-cluster/useDbCluster');
// TODO schedules: this test was written for v1 code and needs to be deleted or rewritten for v2
// vi.mock('hooks/api/backups/useBackups', () => ({
//   useDbBackups: () => ({
//     data: [],
//     isFetching: false,
//   }),
// }));
vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({
    canCreate: true,
  }),
}));

// const storagesMocks = vi.hoisted(() => ({
//   useBackupStoragesByNamespace: vi.fn().mockReturnValue({
//     data: [],
//     isLoading: false,
//   }),
// }));

// vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
//   BACKUP_STORAGES_QUERY_KEY: 'backup-storages',
//   useBackupStoragesByNamespace: storagesMocks.useBackupStoragesByNamespace,
//   useCreateBackupStorage: vi.fn().mockReturnValue({
//     mutate: vi.fn((_, options) => options.onSuccess()),
//     isPending: false,
//   }),
// }));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
    },
  },
});

const FormProviderWrapper = ({
  children,
  values,
}: {
  children: React.ReactNode;
  values?: Partial<DbWizardType>;
}) => {
  const methods = useForm({
    defaultValues: {
      dbType: DbType.Postresql,
      backupsEnabled: true,
      storageLocation: 'S3',
      selectedTime: TimeValue.hours,
      k8sNamespace: 'test',
      minute: 0,
      hour: 12,
      amPm: 'AM',
      weekDay: 'Monday',
      onDay: 1,
      schedules: [],
      ...values,
    },
  });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

describe.skip('BackupsStep', () => {
  it('should render nothing when no backup storage', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Backups />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );

    expect(screen.getByTestId('no-storage-message')).toBeInTheDocument();
    expect(screen.queryByTestId('editable-item')).not.toBeInTheDocument();
  });

  it.each(
    Array(PG_SLOTS_LIMIT)
      .fill(0)
      .map((_, idx) => [idx, idx])
  )(
    'should enforce storage creation when %s schedules and %s storages are in use for PG',
    async (_nrSchedules, _nrStorages) => {
      backupStoragesMocks.useBackupStoragesByNamespace.mockReturnValueOnce({
        data: Array(_nrStorages)
          .fill(0)
          .map((_, idx) => ({
            name: `storage-${idx}`,
            type: StorageType.S3,
            bucketName: `bucket-${idx}`,
            region: 'us-east-1',
          })),
        isLoading: false,
      });
      render(
        <QueryClientProvider client={queryClient}>
          <TestWrapper>
            <FormProviderWrapper
              values={{
                schedules: Array(_nrSchedules)
                  .fill(0)
                  .map((_, idx) => ({
                    backupStorageName: `storage-${idx}`,
                    enabled: true,
                    name: `schedule-${idx}`,
                    schedule: '',
                  })),
              }}
            >
              <Backups />
            </FormProviderWrapper>
          </TestWrapper>
        </QueryClientProvider>
      );

      if (_nrStorages < PG_SLOTS_LIMIT) {
        await waitFor(() =>
          expect(screen.getByTestId('no-storage-message')).toBeInTheDocument()
        );
      } else {
        await waitFor(() =>
          expect(
            screen.queryByTestId('no-storage-message')
          ).not.toBeInTheDocument()
        );
        expect(screen.getByTestId('create-schedule')).toBeDisabled();
      }
    }
  );
  it('should display the Create backup schedule link', async () => {
    backupStoragesMocks.useBackupStoragesByNamespace.mockReturnValue({
      data: [
        {
          bucketName: 'bs-tds-1',
          description: 'bs-tds-1',
          forcePathStyle: false,
          name: 'bs-tds-1',
          namespace: 'everest',
          region: 'us-east-1',
          type: 's3',
          url: 'https://minio.minio.svc.cluster.local',
          verifyTLS: false,
        },
      ],
      isFetching: false,
    });
    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Backups />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );
    await waitFor(() =>
      expect(screen.queryByTestId('no-storage-message')).not.toBeInTheDocument()
    );
    const CreateButton = screen.getByTestId('create-schedule');
    expect(CreateButton).toBeInTheDocument();
  });
  it('should not display the Create backup schedule link if more backup storages are not available', () => {
    backupStoragesMocks.useBackupStoragesByNamespace.mockReturnValue({
      data: [],
      isFetching: false,
    });
    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Backups />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );
    const CreateButton = screen.queryByTestId('create-schedule');
    expect(CreateButton).not.toBeInTheDocument();
  });

  // it('should render everything when backups are enabled', () => {
  //   render(
  //     <TestWrapper>
  //       <FormProviderWrapper>
  //         <ThirdStep />
  //       </FormProviderWrapper>
  //     </TestWrapper>
  //   );

  //   expect(
  //     screen.getByTestId('switch-input-backups-enabled')
  //   ).toBeInTheDocument();
  //   expect(
  //     screen.getByTestId('text-input-storage-location')
  //   ).toBeInTheDocument();
  //   // expect(screen.getByTestId('switch-input-pitr-enabled')).toBeInTheDocument();
  //   // expect(screen.getByTestId('pitr-time-minutes')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-selected-time')).toBeInTheDocument();
  // });

  // //   expect(
  // //     screen.getByTestId('switch-input-backups-enabled')
  // //   ).toBeInTheDocument();
  // //   expect(screen.getByTestId('switch-input-pitr-enabled')).toBeInTheDocument();
  // //   expect(screen.getByTestId('pitr-time-minutes')).toBeInTheDocument();
  // // });

  // //   expect(
  // //     screen.getByTestId('switch-input-backups-enabled')
  // //   ).toBeInTheDocument();
  // //   expect(screen.getByTestId('switch-input-pitr-enabled')).toBeInTheDocument();
  // //   expect(screen.getByTestId('pitr-time-minutes')).toBeInTheDocument();

  // //   fireEvent.click(screen.getByTestId('switch-input-pitr-enabled'));
  // //   expect(screen.queryByTestId('pitr-time-minutes')).not.toBeInTheDocument();
  // // });

  // it('should render hours related field when clicked on hours field', () => {
  //   render(
  //     <TestWrapper>
  //       <FormProviderWrapper>
  //         <ThirdStep />
  //       </FormProviderWrapper>
  //     </TestWrapper>
  //   );

  //   expect(
  //     screen.getByTestId('switch-input-backups-enabled')
  //   ).toBeInTheDocument();
  //   const selectTimeValue = screen.getByTestId('select-input-selected-time');
  //   expect(selectTimeValue).toBeInTheDocument();

  //   fireEvent.change(selectTimeValue, { target: { value: 'hours' } });

  //   expect(selectTimeValue.getAttribute('value')).toBe('hours');
  //   expect(screen.getByTestId('select-input-minute-hour')).toBeInTheDocument();

  //   expect(screen.queryByTestId('select-input-on-day')).not.toBeInTheDocument();
  //   expect(screen.queryByTestId('select-input-hour')).not.toBeInTheDocument();
  //   expect(screen.queryByTestId('select-input-minute')).not.toBeInTheDocument();
  //   expect(screen.queryByTestId('select-input-am-pm')).not.toBeInTheDocument();
  // });

  // it('should render days related field when clicked on days field', () => {
  //   render(
  //     <TestWrapper>
  //       <FormProviderWrapper>
  //         <ThirdStep />
  //       </FormProviderWrapper>
  //     </TestWrapper>
  //   );

  //   expect(
  //     screen.getByTestId('switch-input-backups-enabled')
  //   ).toBeInTheDocument();
  //   const selectTimeValue = screen.getByTestId('select-input-selected-time');
  //   expect(selectTimeValue).toBeInTheDocument();

  //   fireEvent.change(selectTimeValue, { target: { value: 'days' } });

  //   expect(selectTimeValue.getAttribute('value')).toBe('days');

  //   expect(screen.getByTestId('select-input-hour')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-minute')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-am-pm')).toBeInTheDocument();

  //   expect(
  //     screen.queryByTestId('select-input-minute-hour')
  //   ).not.toBeInTheDocument();
  //   expect(
  //     screen.queryByTestId('select-input-week-day')
  //   ).not.toBeInTheDocument();
  //   expect(screen.queryByTestId('select-input-on-day')).not.toBeInTheDocument();
  // });

  // it('should render weeks related field when clicked on weeks field', () => {
  //   render(
  //     <TestWrapper>
  //       <FormProviderWrapper>
  //         <ThirdStep />
  //       </FormProviderWrapper>
  //     </TestWrapper>
  //   );

  //   expect(
  //     screen.getByTestId('switch-input-backups-enabled')
  //   ).toBeInTheDocument();
  //   const selectTimeValue = screen.getByTestId('select-input-selected-time');
  //   expect(selectTimeValue).toBeInTheDocument();

  //   fireEvent.change(selectTimeValue, { target: { value: 'weeks' } });

  //   expect(selectTimeValue.getAttribute('value')).toBe('weeks');

  //   expect(screen.getByTestId('select-input-week-day')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-hour')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-minute')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-am-pm')).toBeInTheDocument();

  //   expect(
  //     screen.queryByTestId('select-input-minute-hour')
  //   ).not.toBeInTheDocument();
  //   expect(screen.queryByTestId('select-input-on-day')).not.toBeInTheDocument();
  // });

  // it('should render months related field when clicked on months field', () => {
  //   render(
  //     <TestWrapper>
  //       <FormProviderWrapper>
  //         <ThirdStep />
  //       </FormProviderWrapper>
  //     </TestWrapper>
  //   );

  //   expect(
  //     screen.getByTestId('switch-input-backups-enabled')
  //   ).toBeInTheDocument();
  //   const selectTimeValue = screen.getByTestId('select-input-selected-time');
  //   expect(selectTimeValue).toBeInTheDocument();

  //   fireEvent.change(selectTimeValue, { target: { value: 'months' } });

  //   expect(selectTimeValue.getAttribute('value')).toBe('months');

  //   expect(screen.getByTestId('select-input-on-day')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-hour')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-minute')).toBeInTheDocument();
  //   expect(screen.getByTestId('select-input-am-pm')).toBeInTheDocument();

  //   expect(
  //     screen.queryByTestId('select-input-minute-hour')
  //   ).not.toBeInTheDocument();
  //   expect(
  //     screen.queryByTestId('select-input-week-day')
  //   ).not.toBeInTheDocument();
  // });
});
