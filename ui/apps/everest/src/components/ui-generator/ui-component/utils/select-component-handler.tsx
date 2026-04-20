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

import { MenuItem } from '@mui/material';
import React from 'react';
import {
  FieldType,
  Component,
  SelectFieldParams,
} from '../../ui-generator.types';
import { Provider } from 'shared-types/api.types';
import { getValueByPath } from './get-value-by-path';

type SelectComponent = Extract<Component, { uiType: FieldType.Select }>;

export const isSelectComponent = (item: Component): item is SelectComponent => {
  return item.uiType === FieldType.Select;
};

export const resolveSelectOptions = (
  selectParams: SelectFieldParams,
  providerObject?: Provider
): { label: string; value: string }[] => {
  if ('options' in selectParams && selectParams.options) {
    return selectParams.options;
  }

  if (
    'optionsPath' in selectParams &&
    selectParams.optionsPath &&
    selectParams.optionsPathConfig &&
    providerObject
  ) {
    const { optionsPath, optionsPathConfig } = selectParams;
    const rawData = getValueByPath(providerObject, optionsPath);

    if (Array.isArray(rawData)) {
      const { labelPath, valuePath } = optionsPathConfig;
      return rawData.map((item) => ({
        label: String(
          getValueByPath(item as Record<string, unknown>, labelPath) ?? ''
        ),
        value: String(
          getValueByPath(item as Record<string, unknown>, valuePath) ?? ''
        ),
      }));
    }
  }

  return [];
};

export const shouldInjectEmptyOption = (
  item: Component,
  options: { label: string; value: string }[]
): boolean => {
  if (!isSelectComponent(item)) return false;

  const isOptional = !item.validation?.required;
  const hasDisplayEmpty = !!item.fieldParams.displayEmpty;
  const hasEmptyOption = options?.some((opt) => opt.value === '');

  return isOptional && hasDisplayEmpty && !hasEmptyOption;
};

export const renderSelectOptions = (
  item: Component,
  name: string,
  providerObject?: Provider
): React.ReactNode[] | undefined => {
  if (!isSelectComponent(item)) return undefined;

  const options = resolveSelectOptions(item.fieldParams, providerObject);
  const optionsNodes: React.ReactNode[] = [];

  if (shouldInjectEmptyOption(item, options)) {
    optionsNodes.push(
      <MenuItem key={`${name}-empty`} value="">
        None
      </MenuItem>
    );
  }

  options.forEach((option) => {
    optionsNodes.push(
      <MenuItem key={`${name}-${option.value}`} value={option.value}>
        {option.label}
      </MenuItem>
    );
  });

  return optionsNodes;
};
