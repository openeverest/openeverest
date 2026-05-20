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

import { InputAdornment } from '@mui/material';
import { TextFieldProps, SelectProps } from '@mui/material';
import { FieldType } from 'components/ui-generator/ui-generator.types';
import { MappedFieldProps } from './get-mapped-params';
import { coerceNumberInputValue } from './coerce-number-input-value';

export const buildFieldProps = (
  uiType: FieldType,
  mappedProps: MappedFieldProps,
  isDisabled: boolean
): Record<string, unknown> => {
  const { badge, textFieldProps, selectFieldProps, ...restMappedProps } =
    mappedProps;

  const finalTextFieldProps = buildTextFieldProps(
    textFieldProps,
    isDisabled,
    badge,
    uiType
  );
  const finalSelectFieldProps = buildSelectFieldProps(
    selectFieldProps,
    isDisabled,
    badge,
    uiType
  );

  return {
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
};

const buildTextFieldProps = (
  textFieldProps: Partial<TextFieldProps> | undefined,
  isDisabled: boolean,
  badge: string | undefined,
  uiType: FieldType
): Partial<TextFieldProps> | undefined => {
  if (!textFieldProps) return undefined;

  const base = {
    ...textFieldProps,
    ...(isDisabled ? { disabled: true } : {}),
  };

  if (badge && uiType === FieldType.Number) {
    return {
      ...base,
      InputProps: {
        ...base.InputProps,
        endAdornment: <InputAdornment position="end">{badge}</InputAdornment>,
      },
    };
  }

  return base;
};

const buildSelectFieldProps = (
  selectFieldProps: Partial<SelectProps> | undefined,
  isDisabled: boolean,
  badge: string | undefined,
  uiType: FieldType
): Partial<SelectProps> | undefined => {
  if (!selectFieldProps) return undefined;

  const base = {
    ...selectFieldProps,
    ...(isDisabled ? { disabled: true } : {}),
  };

  if (badge && uiType === FieldType.Select) {
    return {
      ...base,
      endAdornment: <InputAdornment position="end">{badge}</InputAdornment>,
    };
  }

  return base;
};
