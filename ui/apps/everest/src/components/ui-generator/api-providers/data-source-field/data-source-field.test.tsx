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
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { DataSourceField } from './data-source-field';
import type { ComponentWithDataSource } from './data-source-field.types';
import { Messages } from './data-source-field.messages';
import type { EmptyStateFallbackProps } from '../types';
import {
  isSelectComponent,
  resolveSelectOptions,
} from '../../ui-component/utils/select-component-handler';
import { type Component, FieldType, FormMode } from '../../ui-generator.types';

const mockUseProviderOptions = vi.fn();
const mockRegistryGet = vi.fn();

const CREATE_CONTEXT = { namespace: 'ns' };
const EDIT_CONTEXT = { namespace: 'ns', formMode: FormMode.Edit };
let mockContext: typeof CREATE_CONTEXT | typeof EDIT_CONTEXT = CREATE_CONTEXT;

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
  useUiGeneratorContext: () => mockContext,
}));

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const makeItem = (
  fieldParams: { label: string; disabled?: boolean } = {
    label: 'Storage class',
  }
): ComponentWithDataSource => ({
  uiType: FieldType.Select,
  path: 'spec.storageClass',
  fieldParams,
  dataSource: { provider: 'storageClasses' },
});

describe('DataSourceField', () => {
  afterEach(() => {
    mockContext = CREATE_CONTEXT;
  });

  const renderWithValue = (
    storedValue: string,
    item: ComponentWithDataSource = makeItem()
  ) => {
    mockUseProviderOptions.mockReturnValue({
      options: [{ label: 'standard', value: 'standard' }],
      isLoading: false,
      error: null,
      isEmpty: false,
    });

    let getValues: (name: string) => unknown = () => undefined;

    const Harness = () => {
      const methods = useForm({
        defaultValues: { spec: { storageClass: storedValue } },
      });
      getValues = methods.getValues;
      return (
        <FormProvider {...methods}>
          <DataSourceField item={item} name="spec.storageClass">
            {() => <div />}
          </DataSourceField>
        </FormProvider>
      );
    };

    render(<Harness />);
    return () => getValues('spec.storageClass');
  };

  it('auto-selects a disabled field outside edit mode', async () => {
    const readValue = renderWithValue(
      '',
      makeItem({ label: 'Storage class', disabled: true })
    );

    await waitFor(() => expect(readValue()).toBe('standard'));
  });

  describe('stored value missing from the options', () => {
    const LOADED_OPTIONS = {
      options: [{ label: 'standard', value: 'standard' }],
      isLoading: false,
      error: null,
      isEmpty: false,
    };

    const renderOptionLabels = (patched: Component) =>
      isSelectComponent(patched) ? (
        <>
          <ul>
            {resolveSelectOptions(patched.fieldParams).map((option) => (
              <li key={option.value}>{option.label}</li>
            ))}
          </ul>
          <p data-testid="helper-text">{patched.fieldParams.helperText}</p>
        </>
      ) : null;

    const renderStoredValue = (providerOptions: typeof LOADED_OPTIONS) => {
      mockUseProviderOptions.mockReturnValue(providerOptions);
      let setFormValue: (value: string) => void = () => {};

      const Harness = () => {
        const methods = useForm({
          defaultValues: { spec: { storageClass: 'deleted-class' } },
        });
        setFormValue = (value) => methods.setValue('spec.storageClass', value);
        return (
          <FormProvider {...methods}>
            <DataSourceField item={makeItem()} name="spec.storageClass">
              {renderOptionLabels}
            </DataSourceField>
          </FormProvider>
        );
      };

      const { rerender } = render(<Harness />);
      return {
        pickAndRerender: (value: string) => {
          act(() => setFormValue(value));
          rerender(<Harness />);
        },
      };
    };

    const optionLabels = () =>
      screen.getAllByRole('listitem').map((item) => item.textContent);

    it('lists it as-is in edit mode while the options are loading', () => {
      mockContext = EDIT_CONTEXT;
      renderStoredValue({ ...LOADED_OPTIONS, options: [], isLoading: true });

      expect(optionLabels()).toEqual(['deleted-class']);
    });

    it('keeps listing it after the user picks another option, so the pick can be undone', () => {
      mockContext = EDIT_CONTEXT;
      const { pickAndRerender } = renderStoredValue(LOADED_OPTIONS);

      pickAndRerender('standard');

      expect(optionLabels()).toEqual([
        'standard',
        Messages.unavailableOption('deleted-class'),
      ]);
    });

    it('explains the stored value until the user picks another option', () => {
      mockContext = EDIT_CONTEXT;
      const { pickAndRerender } = renderStoredValue(LOADED_OPTIONS);
      const helperText = screen.getByTestId('helper-text');

      expect(helperText).toHaveTextContent(Messages.unavailableValueHint);

      pickAndRerender('standard');

      expect(helperText).toBeEmptyDOMElement();
    });

    it('does not list it outside edit mode', () => {
      renderStoredValue(LOADED_OPTIONS);

      expect(optionLabels()).toEqual(['standard']);
    });
  });

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
      expect(screen.getByTestId('child')).not.toBeVisible();
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

    it('selects the resource created from the fallback in edit mode', () => {
      mockContext = EDIT_CONTEXT;
      mockUseProviderOptions.mockReturnValue({
        options: [],
        isLoading: false,
        error: null,
        isEmpty: true,
      });
      const CreatingFallback = ({ onCreated }: EmptyStateFallbackProps) => (
        <button type="button" onClick={() => onCreated?.('pmm')}>
          create
        </button>
      );
      mockRegistryGet.mockReturnValue({
        emptyStateFallback: { component: CreatingFallback },
      });
      const monitoringPath =
        'spec.components.monitoring.parameters.monitoringConfigName';
      const monitoringItem: ComponentWithDataSource = {
        uiType: FieldType.Select,
        path: monitoringPath,
        fieldParams: { label: 'Monitoring endpoint' },
        dataSource: { provider: 'monitoringConfigs' },
      };

      let getValues: (name: string) => unknown = () => undefined;
      let isDirty = false;

      const Harness = () => {
        const methods = useForm({
          defaultValues: {
            spec: {
              components: {
                monitoring: { parameters: { monitoringConfigName: '' } },
              },
            },
          },
        });
        getValues = methods.getValues;
        isDirty = methods.formState.isDirty;
        return (
          <FormProvider {...methods}>
            <DataSourceField item={monitoringItem} name={monitoringPath}>
              {() => <div />}
            </DataSourceField>
          </FormProvider>
        );
      };

      render(<Harness />);
      fireEvent.click(screen.getByRole('button', { name: 'create' }));

      expect(getValues(monitoringPath)).toBe('pmm');
      expect(isDirty).toBe(true);
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
