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

import { render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { DataSourceField } from './data-source-field';
import type { ComponentWithDataSource } from './data-source-field.types';
import { FieldType } from '../../ui-generator.types';

const mockUseProviderOptions = vi.fn();
const mockRegistryGet = vi.fn();

vi.mock('../registry', () => ({
  useProviderOptions: (...args: unknown[]) => mockUseProviderOptions(...args),
  providerRegistry: {
    get: (...args: unknown[]) => mockRegistryGet(...args),
    has: () => false,
    getAll: () => new Map(),
    getAvailableKeys: () => [],
  },
}));

vi.mock('../../ui-generator-context', () => ({
  useUiGeneratorContext: () => ({ namespace: 'ns' }),
}));

vi.mock('hooks/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const makeItem = (): ComponentWithDataSource => ({
  uiType: FieldType.Select,
  path: 'spec.storageClass',
  fieldParams: { label: 'Storage class' },
  dataSource: { provider: 'storageClasses' },
});

describe('DataSourceField', () => {
  it('sets form value to first option when options arrive and field is empty', async () => {
    mockUseProviderOptions.mockReturnValue({
      options: [
        { label: 'standard', value: 'standard' },
        { label: 'premium', value: 'premium' },
      ],
      isLoading: false,
      error: null,
      isEmpty: false,
    });

    let getValues: (name: string) => unknown = () => undefined;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: '' } },
      });
      getValues = methods.getValues;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={makeItem()} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    render(<Harness />);

    await waitFor(() => {
      expect(getValues('spec.storageClass')).toBe('standard');
    });
  });

  it('does not overwrite an existing form value', async () => {
    mockUseProviderOptions.mockReturnValue({
      options: [
        { label: 'standard', value: 'standard' },
        { label: 'premium', value: 'premium' },
      ],
      isLoading: false,
      error: null,
      isEmpty: false,
    });

    let getValues: (name: string) => unknown = () => undefined;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: 'premium' } },
      });
      getValues = methods.getValues;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={makeItem()} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    render(<Harness />);

    await waitFor(() => {
      expect(getValues('spec.storageClass')).toBe('premium');
    });
  });

  it('resets form value when current value is no longer in options', async () => {
    const { rerender } = render(<></>);

    mockUseProviderOptions.mockReturnValue({
      options: [
        { label: 'standard', value: 'standard' },
        { label: 'premium', value: 'premium' },
      ],
      isLoading: false,
      error: null,
      isEmpty: false,
    });

    let getValues: (name: string) => unknown = () => undefined;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: 'old-value' } },
      });
      getValues = methods.getValues;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={makeItem()} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    rerender(<Harness />);

    await waitFor(() => {
      expect(getValues('spec.storageClass')).toBe('standard');
    });
  });

  it('clears form value when options become empty', async () => {
    mockUseProviderOptions.mockReturnValue({
      options: [],
      isLoading: false,
      error: null,
      isEmpty: true,
    });

    let getValues: (name: string) => unknown = () => undefined;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: 'stale-value' } },
      });
      getValues = methods.getValues;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={makeItem()} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    render(<Harness />);

    await waitFor(() => {
      expect(getValues('spec.storageClass')).toBe('');
    });
  });

  it('does not mark form dirty when auto-selecting the first option', async () => {
    mockUseProviderOptions.mockReturnValue({
      options: [
        { label: 'standard', value: 'standard' },
        { label: 'premium', value: 'premium' },
      ],
      isLoading: false,
      error: null,
      isEmpty: false,
    });

    let isDirty = true;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: '' } },
      });
      isDirty = methods.formState.isDirty;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={makeItem()} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    render(<Harness />);

    await waitFor(() => {
      expect(isDirty).toBe(false);
    });
  });

  describe('empty state fallback', () => {
    const FallbackStub = ({ namespace }: { namespace: string }) => (
      <div data-testid="fallback">Fallback for {namespace}</div>
    );

    it('renders fallback when provider has emptyStateFallback and options are empty', () => {
      mockUseProviderOptions.mockReturnValue({
        options: [],
        isLoading: false,
        error: null,
        isEmpty: true,
      });
      mockRegistryGet.mockReturnValue({
        emptyStateFallback: { component: FallbackStub },
      });

      const Harness = () => {
        const methods = useForm({
          defaultValues: { spec: { storageClass: '' } },
        });
        return (
          <FormProvider {...methods}>
            <DataSourceField item={makeItem()} name="spec.storageClass">
              {() => <div data-testid="child" />}
            </DataSourceField>
          </FormProvider>
        );
      };

      render(<Harness />);

      expect(screen.getByTestId('fallback')).toBeInTheDocument();
      expect(screen.queryByTestId('child')).not.toBeInTheDocument();
    });

    it('does not render fallback when emptyStateFallback is null', () => {
      mockUseProviderOptions.mockReturnValue({
        options: [],
        isLoading: false,
        error: null,
        isEmpty: true,
      });
      mockRegistryGet.mockReturnValue({ emptyStateFallback: null });

      const Harness = () => {
        const methods = useForm({
          defaultValues: { spec: { storageClass: '' } },
        });
        return (
          <FormProvider {...methods}>
            <DataSourceField item={makeItem()} name="spec.storageClass">
              {() => <div data-testid="child" />}
            </DataSourceField>
          </FormProvider>
        );
      };

      render(<Harness />);

      expect(screen.queryByTestId('fallback')).not.toBeInTheDocument();
      expect(screen.getByTestId('child')).toBeInTheDocument();
    });

    it('does not render fallback when options are loading', () => {
      mockUseProviderOptions.mockReturnValue({
        options: [],
        isLoading: true,
        error: null,
        isEmpty: false,
      });
      mockRegistryGet.mockReturnValue({
        emptyStateFallback: { component: FallbackStub },
      });

      const Harness = () => {
        const methods = useForm({
          defaultValues: { spec: { storageClass: '' } },
        });
        return (
          <FormProvider {...methods}>
            <DataSourceField item={makeItem()} name="spec.storageClass">
              {() => <div data-testid="child" />}
            </DataSourceField>
          </FormProvider>
        );
      };

      render(<Harness />);

      expect(screen.queryByTestId('fallback')).not.toBeInTheDocument();
      expect(screen.getByTestId('child')).toBeInTheDocument();
    });

    it('does not render fallback when options are present', () => {
      mockUseProviderOptions.mockReturnValue({
        options: [{ label: 'standard', value: 'standard' }],
        isLoading: false,
        error: null,
        isEmpty: false,
      });
      mockRegistryGet.mockReturnValue({
        emptyStateFallback: { component: FallbackStub },
      });

      const Harness = () => {
        const methods = useForm({
          defaultValues: { spec: { storageClass: '' } },
        });
        return (
          <FormProvider {...methods}>
            <DataSourceField item={makeItem()} name="spec.storageClass">
              {() => <div data-testid="child" />}
            </DataSourceField>
          </FormProvider>
        );
      };

      render(<Harness />);

      expect(screen.queryByTestId('fallback')).not.toBeInTheDocument();
      expect(screen.getByTestId('child')).toBeInTheDocument();
    });
  });
});
