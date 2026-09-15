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

import { render, screen } from '@testing-library/react';
import {
  resolveSelectOptions,
  shouldInjectEmptyOption,
  renderSelectOptions,
} from '../select-component-handler';
import {
  Component,
  FieldType,
  SelectFieldParams,
} from '../../../ui-generator.types';
import { TestWrapper } from 'utils/test';

const buildSelectComponent = (
  fieldParams: SelectFieldParams,
  required = false
): Extract<Component, { uiType: FieldType.Select }> => ({
  uiType: FieldType.Select,
  path: 'spec.engine',
  fieldParams: {
    label: 'Engine',
    ...fieldParams,
  },
  ...(required ? { validation: { required: true } } : {}),
});

describe('select-component-handler utils', () => {
  it('resolves options from provider object path', () => {
    const options = resolveSelectOptions(
      {
        label: 'Version',
        optionsPath: 'spec.availableVersions.engine',
        optionsPathConfig: {
          labelPath: 'version',
          valuePath: 'version',
          defaultPath: 'default',
        },
      },
      {
        spec: {
          availableVersions: {
            engine: [{ version: '8.0' }, { version: '8.1', default: true }],
          },
        },
      } as never
    );

    expect(options).toEqual([
      { label: '8.0', value: '8.0', isDefault: false,},
      { label: '8.1', value: '8.1', isDefault: true, },
    ]);
  });

  it('injects empty option only for optional + displayEmpty selects', () => {
    const optionalDisplayEmpty = buildSelectComponent({
      options: [{ label: 'MongoDB', value: 'psmdb' }],
      displayEmpty: true,
    });

    const requiredDisplayEmpty = buildSelectComponent(
      {
        options: [{ label: 'MongoDB', value: 'psmdb' }],
        displayEmpty: true,
      },
      true
    );

    expect(
      shouldInjectEmptyOption(optionalDisplayEmpty, [
        { label: 'MongoDB', value: 'psmdb' },
      ])
    ).toBe(true);

    expect(
      shouldInjectEmptyOption(requiredDisplayEmpty, [
        { label: 'MongoDB', value: 'psmdb' },
      ])
    ).toBe(false);
  });

  it('renders menu items including injected empty option', () => {
    const component = buildSelectComponent({
      options: [{ label: 'MongoDB', value: 'psmdb' }],
      displayEmpty: true,
    });

    const nodes = renderSelectOptions(component, 'spec.engine');

    render(<TestWrapper>{nodes}</TestWrapper>);

    expect(screen.getByText('None')).toBeInTheDocument();
    expect(screen.getByText('MongoDB')).toBeInTheDocument();
  });
});
