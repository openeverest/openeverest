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

import { render, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { DataSourceField } from './data-source-field';
import type { ComponentWithDataSource } from './data-source-field.types';
import { FieldType } from '../../ui-generator.types';

const mockUseProviderOptions = vi.fn();

vi.mock('../registry', () => ({
  useProviderOptions: (...args: unknown[]) => mockUseProviderOptions(...args),
}));

vi.mock('../../ui-generator-context', () => ({
  useUiGeneratorContext: () => ({ namespace: 'ns', cluster: 'cl' }),
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
});
