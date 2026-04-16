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

import { Component } from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { useFormContext, get } from 'react-hook-form';
import { muiComponentMap } from '../constants';
import { renderComponentChildren } from './utils/component-renderer';
import { useUiGeneratorContext } from '../ui-generator-context';
import { getMappedParams, MappedFieldProps } from './utils/get-mapped-params';
import { resolveValidationForMode } from '../utils/validation/resolve-validation-for-mode';
import { buildFieldProps } from './utils/build-field-props';
import { applyFieldWrappers } from './utils/field-wrappers';

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

const UIComponent: React.FC<ComponentProps> = ({ item, name }) => {
  const { uiType, fieldParams, validation } = item;
  const methods = useFormContext();
  const errors = methods?.formState?.errors || {};
  const { providerObject, loadingDefaultsForEdition, formMode } =
    useUiGeneratorContext();

  const isDisabled = !!loadingDefaultsForEdition || !!fieldParams?.disabled;
  const resolvedValidation = resolveValidationForMode(validation, formMode);
  const errorObj = get(errors, name);
  const error = errorObj?.message as string | undefined;

  const MuiComponent = muiComponentMap[uiType];
  const label = fieldParams?.label || '';

  if (!MuiComponent) return null;

  const mappedProps = getMappedParams(uiType, fieldParams, resolvedValidation);
  const finalProps = buildFieldProps(
    uiType,
    mappedProps as MappedFieldProps,
    isDisabled
  );

  const children = renderComponentChildren(item, name, providerObject);

  const fieldElement = React.createElement(
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
  );

  return <>{applyFieldWrappers(fieldElement, item)}</>;
};

export default UIComponent;
