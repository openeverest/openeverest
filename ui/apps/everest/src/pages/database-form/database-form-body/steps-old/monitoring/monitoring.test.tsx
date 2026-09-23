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

import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { Monitoring } from './monitoring';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WizardMode } from 'shared-types/wizard.types';

const mocks = vi.hoisted(() => {
  return {
    useMonitoringInstancesList: vi.fn().mockReturnValue([
      {
        namespace: 'the-dark-side',
        queryResult: {
          status: 'success',
          fetchStatus: 'fetching',
          isPending: false,
          isSuccess: true,
          isError: false,
          isLoading: false,
          data: [
            {
              allowedNamespaces: null,
              name: 'PMM-local',
              namespace: 'the-dark-side',
              type: 'pmm',
              url: '127.0.0.1',
              verifyTLS: true,
            },
          ],
        },
      },
    ]),
    useCreateMonitoringInstance: vi.fn().mockReturnValue({
      type: 'type1',
      url: '127.0.0.1',
      name: 'PMM-local',
      namespace: 'the-dark-side',
    }),
  };
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
    },
  },
});

vi.mock('hooks/api/monitoring/useMonitoringConfigsList', () => ({
  useMonitoringConfigsList: mocks.useMonitoringInstancesList,
  useCreateMonitoringConfig: mocks.useCreateMonitoringInstance,
  MONITORING_CONFIGS_QUERY_KEY: 'monitoringConfigs',
}));

vi.mock('../../useDatabasePageMode', () => ({
  useDatabasePageMode: () => WizardMode.New,
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({
    canCreate: true,
  }),
}));

const FormProviderWrapper = ({ children }: { children: React.ReactNode }) => {
  const methods = useForm({
    defaultValues: {
      monitoring: false,
      monitoringInstance: '',
      k8sNamespace: 'the-dark-side',
    },
  });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

describe.skip('Monitoring Step', () => {
  it("should render only monitoring input if it's off", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Monitoring />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );
    expect(screen.getByTestId('switch-input-monitoring')).toBeInTheDocument();
    expect(screen.getByRole('checkbox')).not.toBeDisabled();
    expect(screen.queryByTestId('monitoring-warning')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('text-input-monitoring-instance')
    ).not.toBeInTheDocument();
  });

  it('should render remaining fields when monitoring is on', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Monitoring />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );

    expect(screen.getByTestId('switch-input-monitoring')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('switch-input-monitoring'));
    await waitFor(() =>
      expect(
        screen.getByTestId('text-input-monitoring-instance')
      ).toBeInTheDocument()
    );
  });

  it('should disable toggle when no monitoring instances defined', async () => {
    mocks.useMonitoringInstancesList.mockReturnValue([]);

    render(
      <QueryClientProvider client={queryClient}>
        <TestWrapper>
          <FormProviderWrapper>
            <Monitoring />
          </FormProviderWrapper>
        </TestWrapper>
      </QueryClientProvider>
    );

    expect(screen.getByRole('checkbox')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('checkbox')).toBeDisabled());
  });
});
