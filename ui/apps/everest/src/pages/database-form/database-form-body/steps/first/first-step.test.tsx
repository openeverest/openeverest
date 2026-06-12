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

import { ReactNode } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { DbEngineType, DbType } from '@percona/types';
import { TestWrapper } from 'utils/test';
import { FirstStep } from './first-step';
import { DbWizardFormFields } from 'consts';
import {
  DbEngineStatus,
  DbEngineToolStatus,
} from 'shared-types/dbEngines.types';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const mocks = vi.hoisted(() => {
  return {
    getDbEnginesFn: vi.fn(() =>
      Promise.resolve({
        items: [
          {
            spec: { type: DbEngineType.PXC },
            status: {
              status: DbEngineStatus.INSTALLED,
              availableVersions: {
                engine: {
                  '8.0': {
                    version: '8.0',
                    description: '8.0',
                    status: DbEngineToolStatus.AVAILABLE,
                  },
                  '8.0.31-23.2': {
                    version: '8.0.31-23.2',
                    description: '8.0.31-23.2',
                    status: DbEngineToolStatus.RECOMMENDED,
                  },
                  '9.0.0': {
                    version: '9.0.0',
                    description: '8.0.31-23.2',
                    status: DbEngineToolStatus.AVAILABLE,
                  },
                  '8.1': {
                    version: '8.1',
                    description: '8.1',
                    status: DbEngineToolStatus.RECOMMENDED,
                  },
                  '8.4': {
                    version: '8.4',
                    description: '8.4',
                    status: DbEngineToolStatus.AVAILABLE,
                  },
                },
              },
            },
            metadata: { name: 'pxc-1' },
          },
        ],
      })
    ),
  };
});

vi.mock('./utils', () => ({
  generateShortUID: vi.fn(() => '123'),
}));

vi.mock('api/kubernetesClusterApi', () => ({
  getKubernetesClusterInfoFn: vi.fn(() =>
    Promise.resolve({
      clusterType: 'generic',
    })
  ),
}));

vi.mock('api/namespaces', () => ({
  getNamespacesFn: vi.fn(() => Promise.resolve(['namespace-1'])),
}));

vi.mock('api/dbEngineApi', () => ({
  getDbEnginesFn: mocks.getDbEnginesFn,
}));

interface FormProviderWrapperProps {
  handleSubmit: () => void;
  children: ReactNode;
}
const FormProviderWrapper = ({
  children,
  handleSubmit,
}: FormProviderWrapperProps) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
      },
    },
  });
  const methods = useForm({
    defaultValues: {
      [DbWizardFormFields.dbType]: DbType.Mysql,
      [DbWizardFormFields.dbName]: 'mysql-123',
      [DbWizardFormFields.dbVersion]: '8.0',
    },
  });

  return (
    <QueryClientProvider client={queryClient}>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(handleSubmit)}>{children}</form>
      </FormProvider>
    </QueryClientProvider>
  );
};

describe('First Step', async () => {
  it('should show sorted db versions with selected recommended version', async () => {
    render(
      <TestWrapper>
        <FormProviderWrapper handleSubmit={vi.fn()}>
          <FirstStep loadingDefaultsForEdition={false} alreadyVisited={false} />
        </FormProviderWrapper>
      </TestWrapper>
    );

    // 8.1 is the latest recommended version, therefore it should be the selected option after loading
    await waitFor(
      () =>
        expect(screen.getByTestId('select-input-db-version')).toHaveValue(
          '8.1'
        ),
      { timeout: 10000 }
    );

    const dbVersionComboBox = screen
      .getAllByRole('combobox')
      .find((el) => el.id === 'mui-component-select-dbVersion');

    expect(dbVersionComboBox).toBeDefined();

    fireEvent.mouseDown(dbVersionComboBox!);
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    const versionOptions = screen.getAllByRole('option');

    expect(versionOptions).toHaveLength(5);
    expect(versionOptions.map((option) => option.textContent)).toEqual([
      '9.0.0',
      '8.4',
      '8.1',
      '8.0.31-23.2',
      '8.0',
    ]);
  });

  it('should show highest version if recommended version is not available', async () => {
    mocks.getDbEnginesFn.mockResolvedValue({
      items: [
        {
          spec: { type: DbEngineType.PXC },
          status: {
            status: DbEngineStatus.INSTALLED,
            availableVersions: {
              engine: {
                '8.0': {
                  version: '8.0',
                  description: '8.0',
                  status: DbEngineToolStatus.AVAILABLE,
                },
                '8.0.31-23.2': {
                  version: '8.0.31-23.2',
                  description: '8.0.31-23.2',
                  status: DbEngineToolStatus.AVAILABLE,
                },
                '9.0.0': {
                  version: '9.0.0',
                  description: '8.0.31-23.2',
                  status: DbEngineToolStatus.AVAILABLE,
                },
                '8.1': {
                  version: '8.1',
                  description: '8.1',
                  status: DbEngineToolStatus.AVAILABLE,
                },
                '8.4': {
                  version: '8.4',
                  description: '8.4',
                  status: DbEngineToolStatus.AVAILABLE,
                },
              },
            },
          },
          metadata: { name: 'pxc-1' },
        },
      ],
    });

    render(
      <TestWrapper>
        <FormProviderWrapper handleSubmit={vi.fn()}>
          <FirstStep loadingDefaultsForEdition={false} alreadyVisited={false} />
        </FormProviderWrapper>
      </TestWrapper>
    );

    // no version is recommended, therefore 9.0.0 should be the selected option after loading
    await waitFor(
      () =>
        expect(screen.getByTestId('select-input-db-version')).toHaveValue(
          '9.0.0'
        ),
      { timeout: 10000 }
    );
  });
});
