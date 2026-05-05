import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { AdvancedConfigurations } from './advanced-configurations';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { advancedConfigurationsSchema } from 'components/cluster-form/advanced-configuration/advanced-configuration-schema';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { DbWizardType } from 'pages/database-form/database-form-schema';

const queryClient = new QueryClient();

const FormProviderWrapper = ({
  children,
  values,
}: {
  children: React.ReactNode;
  values?: Partial<DbWizardType>;
}) => {
  const methods = useForm({
    mode: 'onChange',
    resolver: zodResolver(advancedConfigurationsSchema()),
    defaultValues: {
      storageClass: 'standard',
      engineParametersEnabled: false,
      podSchedulingPolicyEnabled: false,
      exposureMethod: ProxyExposeType.ClusterIP,
      splitHorizonDNSEnabled: false,
      splitHorizonDNS: '',

      sourceRanges: [
        {
          sourceRange: '192.168.1.1',
        },
      ],
      ...values,
    },
  });

  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(vi.fn())}>{children}</form>
    </FormProvider>
  );
};

describe('FourthStep', () => {
  it('should show an error message when duplicate sourceRanges are added', async () => {
    render(
      <TestWrapper>
        <FormProviderWrapper
          values={{
            sourceRanges: [
              {
                sourceRange: '',
              },
            ],
            exposureMethod: ProxyExposeType.LoadBalancer,
          }}
        >
          <QueryClientProvider client={queryClient}>
            <AdvancedConfigurations
              loadingDefaultsForEdition={false}
              alreadyVisited={false}
            />
          </QueryClientProvider>
        </FormProviderWrapper>
      </TestWrapper>
    );

    fireEvent.mouseDown(
      screen
        .getAllByRole('combobox')
        .find((el) => el.id === 'mui-component-select-exposureMethod')!
    );
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    const loadBalancerOption = screen
      .getAllByRole('option')
      .find((el) => el.textContent === 'Load balancer');

    expect(loadBalancerOption).toBeDefined();
    fireEvent.click(loadBalancerOption!);

    await waitFor(() =>
      expect(screen.getByTestId('select-input-exposure-method')).toHaveValue(
        ProxyExposeType.LoadBalancer
      )
    );

    await waitFor(() =>
      expect(screen.getByTestId('external-access-fields')).toBeInTheDocument()
    );

    // Add the first source range
    const firstSourceRangeInput = screen.getByTestId(
      'text-input-source-ranges.0.source-range'
    );

    fireEvent.input(firstSourceRangeInput, {
      target: { value: '192.168.1.1/32' },
    });
    fireEvent.blur(firstSourceRangeInput);

    await waitFor(() => {
      const addButton = screen.getByTestId('add-text-input-button');
      expect(addButton).not.toBeDisabled();
    });

    // Add a new source range input
    fireEvent.click(screen.getByTestId('add-text-input-button'));

    await waitFor(() => {
      expect(
        screen.getByTestId('text-input-source-ranges.1.source-range')
      ).toBeInTheDocument();
    });

    const secondSourceRangeInput = screen.getByTestId(
      'text-input-source-ranges.1.source-range'
    );

    await waitFor(() =>
      fireEvent.input(secondSourceRangeInput, {
        target: { value: '192.168.1.1/32' },
      })
    );
    fireEvent.input(secondSourceRangeInput, {
      target: { value: '192.168.1.1/32' },
    });
    fireEvent.blur(secondSourceRangeInput);

    await waitFor(() => expect(secondSourceRangeInput).toBeInvalid());

    // Check if the duplicate error message is displayed

    expect(
      screen.getByText(
        'Duplicate entry. This IP and netmask combination already exists.'
      )
    ).toBeInTheDocument();
  });

  it('should trigger validation when switching from Cluster IP to Load Balancer', async () => {
    render(
      <TestWrapper>
        <FormProviderWrapper
          values={{
            sourceRanges: [
              {
                sourceRange: 'invalid-ip',
              },
            ],
            exposureMethod: ProxyExposeType.ClusterIP,
          }}
        >
          <QueryClientProvider client={queryClient}>
            <AdvancedConfigurations
              loadingDefaultsForEdition={false}
              alreadyVisited={false}
            />
          </QueryClientProvider>
        </FormProviderWrapper>
      </TestWrapper>
    );

    expect(
      screen.queryByTestId('external-access-fields')
    ).not.toBeInTheDocument();
    fireEvent.mouseDown(
      screen
        .getAllByRole('combobox')
        .find((el) => el.id === 'mui-component-select-exposureMethod')!
    );
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    const loadBalancerOption = screen
      .getAllByRole('option')
      .find((el) => el.textContent === 'Load balancer');

    expect(loadBalancerOption).toBeDefined();
    fireEvent.click(loadBalancerOption!);

    await waitFor(() =>
      expect(screen.getByTestId('external-access-fields')).toBeInTheDocument()
    );
    const sourceRangeInput = screen.getByTestId(
      'text-input-source-ranges.0.source-range'
    );

    await waitFor(() => {
      expect(sourceRangeInput).toBeInvalid();
    });
  });

  it('should clear duplicate error messages when deleting duplicate source range entries', async () => {
    render(
      <TestWrapper>
        <FormProviderWrapper
          values={{
            sourceRanges: [
              {
                sourceRange: '192.168.1.1/32',
              },
              {
                sourceRange: '192.168.1.1/32',
              },
            ],
            exposureMethod: ProxyExposeType.LoadBalancer,
          }}
        >
          <QueryClientProvider client={queryClient}>
            <AdvancedConfigurations
              loadingDefaultsForEdition={false}
              alreadyVisited={false}
            />
          </QueryClientProvider>
        </FormProviderWrapper>
      </TestWrapper>
    );

    await waitFor(() =>
      expect(screen.getByTestId('external-access-fields')).toBeInTheDocument()
    );

    const firstSourceRangeInput = screen.getByTestId(
      'text-input-source-ranges.0.source-range'
    );
    const secondSourceRangeInput = screen.getByTestId(
      'text-input-source-ranges.1.source-range'
    );

    await waitFor(() => {
      expect(secondSourceRangeInput).toBeInvalid();
    });

    fireEvent.click(screen.getByTestId('delete-text-input-1-button'));

    await waitFor(() => {
      expect(
        screen.queryByTestId('text-input-source-ranges.1.source-range')
      ).not.toBeInTheDocument();
    });
    await waitFor(() => {
      expect(firstSourceRangeInput).not.toBeInvalid();
    });
  });
});
