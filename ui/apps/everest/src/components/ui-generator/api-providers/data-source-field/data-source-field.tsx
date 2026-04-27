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

import React, { useEffect, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import type { Component } from '../../ui-generator.types';
import { useProviderOptions } from '../registry';
import { useUiGeneratorContext } from '../../ui-generator-context';
import type { DataSourceFieldProps } from './data-source-field.types';

export const DataSourceField: React.FC<DataSourceFieldProps> = ({
  item,
  name,
  children,
}) => {
  const { namespace, cluster } = useUiGeneratorContext();
  const { getValues, setValue } = useFormContext();
  const { dataSource, ...baseComponent } = item;

  const { options, isLoading, error, isEmpty } = useProviderOptions(
    dataSource.provider,
    {
      namespace: namespace ?? '',
      cluster: cluster ?? '',
      config: dataSource.config,
    }
  );

  useEffect(() => {
    if (isLoading || options.length === 0 || !name) return;

    const current = getValues(name);
    if (current === '' || current === undefined || current === null) {
      setValue(name, options[0].value, { shouldValidate: true });
    }
  }, [isLoading, options, name, getValues, setValue]);

  const patchedItem = useMemo(() => {
    const originalHelperText = (
      baseComponent.fieldParams as Record<string, unknown>
    )?.helperText;

    const helperText = isLoading
      ? 'Loading...'
      : error
        ? 'Failed to load options'
        : isEmpty
          ? (originalHelperText ?? 'No options available')
          : originalHelperText;

    return {
      ...baseComponent,
      fieldParams: {
        ...baseComponent.fieldParams,
        options,
        disabled:
          isLoading ||
          !!error ||
          (baseComponent.fieldParams as Record<string, unknown>)?.disabled,
        ...(helperText !== undefined ? { helperText } : {}),
      },
    } as Component;
  }, [baseComponent, options, isLoading, error, isEmpty]);

  // TODO: Render fallback component when isEmpty/error and fallback is defined
  // (requires Alert component — see separate issue)

  return <>{children(patchedItem)}</>;
};
