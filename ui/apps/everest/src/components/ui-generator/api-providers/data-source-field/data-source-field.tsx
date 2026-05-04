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

import React, { useEffect, useMemo, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import type { Component } from '../../ui-generator.types';
import { providerRegistry, useProviderOptions } from '../registry';
import { useUiGeneratorContext } from '../../ui-generator-context';
import { useClusterName } from 'hooks/api/useClusterName';
import type { DataSourceFieldProps } from './data-source-field.types';

const resolveFieldValue = (
  current: unknown,
  options: { value: string }[]
): string | null => {
  const validValues = options.map((o) => o.value);
  const currentStr = typeof current === 'string' ? current : '';

  if (currentStr && !validValues.includes(currentStr)) {
    return options.length > 0 ? options[0].value : '';
  }
  if (
    (currentStr === '' || current === undefined || current === null) &&
    options.length > 0
  ) {
    return options[0].value;
  }
  return null;
};

export const DataSourceField: React.FC<DataSourceFieldProps> = ({
  item,
  name,
  children,
}) => {
  const { namespace } = useUiGeneratorContext();
  const cluster = useClusterName();
  const { getValues, setValue } = useFormContext();
  const { dataSource, ...baseComponent } = item;

  const hasValidContext = !!namespace && !!cluster;

  const { options, isLoading, error, isEmpty } = useProviderOptions(
    dataSource.provider,
    {
      namespace: namespace ?? '',
      cluster,
    },
    { enabled: hasValidContext }
  );

  // Synchronous value sync: ensures the form value is correct BEFORE the
  // child Controller renders. This eliminates the timing gap where the
  // Controller would mount with a stale/empty value and then re-render
  // one tick later after the useEffect fires.
  const lastSyncedRef = useRef<{ options: typeof options; name: string }>();

  if (
    !isLoading &&
    name &&
    options.length > 0 &&
    (lastSyncedRef.current?.options !== options ||
      lastSyncedRef.current?.name !== name)
  ) {
    const current = getValues(name);
    const corrected = resolveFieldValue(current, options);
    if (corrected !== null) {
      setValue(name, corrected);
    }
    lastSyncedRef.current = { options, name };
  }

  // Async fallback: handles value invalidation when options change after
  // the component is already mounted (e.g. namespace switch, config deletion).
  useEffect(() => {
    if (isLoading || !name) return;

    const current = getValues(name);
    const corrected = resolveFieldValue(current, options);
    if (corrected !== null) {
      setValue(name, corrected, {
        shouldValidate: true,
      });
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

  const fallback = useMemo(() => {
    const entry = providerRegistry.get(dataSource.provider);
    return entry?.emptyStateFallback ?? null;
  }, [dataSource.provider]);

  if (isEmpty && !isLoading && fallback && namespace) {
    const { component: FallbackComponent } = fallback;

    return <FallbackComponent namespace={namespace} cluster={cluster} />;
  }

  return <>{children(patchedItem)}</>;
};
