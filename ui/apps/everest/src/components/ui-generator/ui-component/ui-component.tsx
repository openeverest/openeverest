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
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { useFormContext, get } from 'react-hook-form';
import { InputAdornment } from '@mui/material';
import { muiComponentMap } from '../constants';
import { renderComponentChildren } from './utils/component-renderer';
import { useUiGeneratorContext } from '../ui-generator-context';
import { getMappedParams, MappedFieldProps } from './utils/get-mapped-params';

type ComponentByType<T extends Component['uiType']> = Extract<
  Component,
  { uiType: T }
>;

export type ComponentProps<
  T extends Component['uiType'] = Component['uiType'],
> = {
  item: ComponentByType<T>;
  name: string;
};

const coerceNumberInputValue = (value: unknown): unknown => {
  if (value === undefined || value === null || value === '') {
    return '';
  }

  if (typeof value === 'number') {
    return value;
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return '';
    }

    const parsed = Number(trimmed);
    return Number.isNaN(parsed) ? value : parsed;
  }

  return value;
};

const UIComponent: React.FC<ComponentProps> = ({ item, name }) => {
  const { uiType, fieldParams, validation } = item;
  const methods = useFormContext();
  const errors = methods?.formState?.errors || {};
  const { providerObject, loadingDefaultsForEdition } = useUiGeneratorContext();
  const isDisabled = !!loadingDefaultsForEdition;

  //get() is used to access nested error paths like "spec.replica.nodes"
  const errorObj = get(errors, name);
  const error = errorObj?.message as string | undefined;

  const MuiComponent = muiComponentMap[uiType];

  const label = fieldParams?.label || '';

  if (!MuiComponent) return null;

  const mappedProps = getMappedParams(uiType, fieldParams, validation);

  // Extract badge from mappedProps if present
  const { badge, textFieldProps, selectFieldProps, ...restMappedProps } =
    mappedProps as MappedFieldProps;

  // Add badge as InputAdornment if present
  let finalTextFieldProps = textFieldProps
    ? { ...textFieldProps, ...(isDisabled ? { disabled: true } : {}) }
    : undefined;
  if (badge && uiType === FieldType.Number && textFieldProps) {
    // For Number fields (TextField-based), add InputProps with endAdornment
    finalTextFieldProps = {
      ...textFieldProps,
      InputProps: {
        ...textFieldProps.InputProps,
        endAdornment: <InputAdornment position="end">{badge}</InputAdornment>,
      },
    };
  }

  // For Select fields, badge handling different - needs to be in selectFieldProps
  let finalSelectFieldProps = selectFieldProps
    ? { ...selectFieldProps, ...(isDisabled ? { disabled: true } : {}) }
    : undefined;
  if (badge && uiType === FieldType.Select && selectFieldProps) {
    finalSelectFieldProps = {
      ...selectFieldProps,
      endAdornment: <InputAdornment position="end">{badge}</InputAdornment>,
    };
  }

  const finalProps = {
    ...(isDisabled ? { disabled: true } : {}),
    ...restMappedProps,
    ...(uiType === FieldType.Number
      ? {
          controllerProps: {
            ...(restMappedProps.controllerProps as Record<string, unknown>),
            rules: {
              ...((restMappedProps.controllerProps as { rules?: object })
                ?.rules ?? {}),
              setValueAs: coerceNumberInputValue,
            },
          },
        }
      : {}),
    ...(finalTextFieldProps ? { textFieldProps: finalTextFieldProps } : {}),
    ...(finalSelectFieldProps
      ? { selectFieldProps: finalSelectFieldProps }
      : {}),
  };

  // Render component-specific children (e.g., MenuItem options for Select)
  const children = renderComponentChildren(item, name, providerObject);

  return (
    <>
      {React.createElement(
        MuiComponent,
        {
          ...finalProps,
          name,
          label,
          error: !!error,
          helperText: error,
          formControlProps: { sx: { minWidth: '450px', marginTop: '15px' } },
        },
        children
      )}
    </>
  );
};

export default UIComponent;
