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

import { useContext } from 'react';
import type { ReactNode } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { DbWizardFormFields } from 'consts.ts';
import { ScheduleFormDialogContext } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog.context';
import type { BackupClass } from 'shared-types/backups.types';
import { Schedules } from './schedules';

const backupClasses: BackupClass[] = [
  {
    metadata: { name: 'psmdb-class' },
    spec: {
      executionMode: 'ProviderManaged',
      supportedProviders: ['psmdb'],
    },
  },
  {
    metadata: { name: 'pxc-class' },
    spec: {
      executionMode: 'ProviderManaged',
      supportedProviders: ['pxc'],
    },
  },
  {
    metadata: { name: 'pxc-job-class' },
    spec: { executionMode: 'Job', supportedProviders: ['pxc'] },
  },
];
const backupClassesResponse = { data: backupClasses, isSuccess: true };

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));
vi.mock('pages/database-form/hooks/use-database-page-mode', () => ({
  useDatabasePageMode: () => 'new',
}));
vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useBackupClassesList: () => backupClassesResponse,
}));
vi.mock('components/editable-item/editable-item', () => ({
  default: () => null,
}));
vi.mock('@percona/ui-lib', () => ({
  ActionableLabeledContent: ({
    actionButtonProps,
    children,
  }: {
    actionButtonProps: {
      disabled: boolean;
      dataTestId: string;
      onClick: () => void;
    };
    children: ReactNode;
  }) => (
    <>
      <button
        data-testid={actionButtonProps.dataTestId}
        disabled={actionButtonProps.disabled}
        onClick={actionButtonProps.onClick}
      />
      {children}
    </>
  ),
}));

const ScheduleDialogProbe = () => {
  const { dbInstanceInfo, handleClose } = useContext(ScheduleFormDialogContext);
  return (
    <>
      <div data-testid="available-classes">
        {dbInstanceInfo.availableBackupClasses
          .map((backupClass) => backupClass.metadata?.name)
          .join(',')}
      </div>
      <button onClick={handleClose}>Close schedule</button>
    </>
  );
};

vi.mock('components/schedule-form-dialog', () => ({
  ScheduleFormDialog: () => <ScheduleDialogProbe />,
}));

const Harness = () => {
  const methods = useForm({
    defaultValues: {
      provider: 'pxc',
      k8sNamespace: 'test',
      dbName: 'db',
      backup: { classRef: { name: 'psmdb-class' }, schedules: [] },
    },
  });
  const selectedClass = useWatch({
    control: methods.control,
    name: 'backup.classRef.name',
  });

  return (
    <FormProvider {...methods}>
      <button
        onClick={() => methods.setValue(DbWizardFormFields.provider, 'psmdb')}
      >
        Select PSMDB
      </button>
      <button
        onClick={() => methods.setValue(DbWizardFormFields.provider, 'unknown')}
      >
        Select unknown
      </button>
      <div data-testid="selected-class">{selectedClass}</div>
      <Schedules backupStorages={[{ spec: { type: 's3' } }]} />
    </FormProvider>
  );
};

it('shows only provider-managed classes for the selected provider', async () => {
  render(<Harness />);

  await waitFor(() =>
    expect(screen.getByTestId('selected-class')).toHaveTextContent('pxc-class')
  );
  fireEvent.click(screen.getByTestId('create-schedule'));
  expect(screen.getByTestId('available-classes')).toHaveTextContent('pxc-class');
  expect(screen.getByTestId('available-classes')).not.toHaveTextContent(
    'psmdb-class'
  );
  expect(screen.getByTestId('available-classes')).not.toHaveTextContent(
    'pxc-job-class'
  );

  fireEvent.click(screen.getByText('Close schedule'));
  fireEvent.click(screen.getByText('Select PSMDB'));
  await waitFor(() =>
    expect(screen.getByTestId('selected-class')).toHaveTextContent(
      'psmdb-class'
    )
  );
  fireEvent.click(screen.getByTestId('create-schedule'));
  expect(screen.getByTestId('available-classes')).toHaveTextContent(
    'psmdb-class'
  );
  expect(screen.getByTestId('available-classes')).not.toHaveTextContent(
    'pxc-class'
  );

  fireEvent.click(screen.getByText('Close schedule'));
  fireEvent.click(screen.getByText('Select unknown'));
  await waitFor(() =>
    expect(screen.getByTestId('selected-class')).toBeEmptyDOMElement()
  );
  expect(screen.getByTestId('create-schedule')).toBeDisabled();
});
