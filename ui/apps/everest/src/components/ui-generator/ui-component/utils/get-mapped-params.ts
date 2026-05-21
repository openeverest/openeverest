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

import { SelectProps, TextFieldProps } from '@mui/material';
import {
  NumberFieldParams,
  SelectFieldParams,
  TextFieldParams,
  FieldParamsMap,
  ValidationMap,
  FieldType,
} from '../../ui-generator.types';

export type MappedFieldProps = {
  badge?: string;
  textFieldProps?: Partial<TextFieldProps>;
  selectFieldProps?: Partial<SelectProps>;
  label?: string;
  defaultValue?: unknown;
  options?: { label: string; value: string }[];
  helperText?: string;
  step?: number;
} & Record<string, unknown>;

// Helper to filter out undefined values from an object
const filterDefined = <T extends Record<string, unknown>>(
  obj: T
): Partial<T> => {
  return Object.fromEntries(
    Object.entries(obj).filter(([, value]) => value !== undefined)
  ) as Partial<T>;
};

export const getMappedParams = <K extends keyof FieldParamsMap>(
  fieldType: K,
  fieldParams: FieldParamsMap[K],
  validation?: ValidationMap[K]
) => {
  switch (fieldType) {
    case 'number':
      return mapNumberFieldParams(
        fieldParams as NumberFieldParams,
        validation as ValidationMap[FieldType.Number] | undefined
      );
    case 'text':
      return mapTextFieldParams(fieldParams as TextFieldParams);
    case 'select':
      return mapSelectFieldParams(fieldParams as SelectFieldParams);
    // Add more cases for other field types as needed
    default:
      return fieldParams;
  }
};

const mapNumberFieldParams = (
  fieldParams: NumberFieldParams,
  validation?: ValidationMap[FieldType.Number]
) => {
  const { disabled, helperText, badge, autoFocus, placeholder, step, ...rest } =
    fieldParams;

  const textFieldProps: Partial<TextFieldProps> = filterDefined({
    type: 'number' as const,
    disabled,
    helperText,
    autoFocus,
    placeholder,
  });

  const getOffset = (): number => {
    if (validation?.int) {
      return 1;
    }
    if (step !== undefined) {
      return step;
    }
    return 0.000001;
  };

  const offset = getOffset();

  // Priority: explicit min/max > converted gt/lt
  const minValue =
    validation?.min !== undefined
      ? validation.min
      : validation?.gt !== undefined
        ? validation.gt + offset
        : undefined;

  const maxValue =
    validation?.max !== undefined
      ? validation.max
      : validation?.lt !== undefined
        ? validation.lt - offset
        : undefined;

  const inputProps = filterDefined({
    min: minValue,
    max: maxValue,
    step,
  });

  if (Object.keys(inputProps).length > 0) {
    textFieldProps.inputProps = inputProps;
  }

  return {
    ...rest,
    textFieldProps: {
      ...textFieldProps,
    },
    badge,
  };
};

const mapSelectFieldParams = (fieldParams: SelectFieldParams) => {
  const { label, defaultValue, options, helperText, badge, ...selectProps } =
    fieldParams;

  const selectFieldProps: Partial<SelectProps> = filterDefined(
    selectProps as Record<string, unknown>
  ) as Partial<SelectProps>;

  return {
    label,
    options,
    helperText,
    selectFieldProps,
    badge,
    controllerProps: { defaultValue: defaultValue ?? '' },
  };
};

const mapTextFieldParams = (fieldParams: TextFieldParams) => {
  const { label, defaultValue, readOnly, badge, ...textProps } = fieldParams;

  const textFieldProps: Partial<TextFieldProps> = filterDefined(
    textProps as Record<string, unknown>
  ) as Partial<TextFieldProps>;

  if (readOnly !== undefined) {
    textFieldProps.InputProps = { readOnly };
  }

  return { label, defaultValue, textFieldProps, badge };
};
