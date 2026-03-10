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

vi.mock('hooks/api/monitoring/useMonitoringInstancesList', () => ({
  useMonitoringInstancesList: mocks.useMonitoringInstancesList,
  useCreateMonitoringInstance: mocks.useCreateMonitoringInstance,
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
