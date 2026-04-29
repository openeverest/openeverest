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

import { useEffect, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import type { Component, ComponentGroup, Section } from '../ui-generator.types';
import { hasDataSource } from './data-source-field';
import { useProviderOptions } from './registry';
import { ComponentErrorBoundary } from '../component-error-boundary';
import { getComponentSourcePath } from '../utils/preprocess/normalized-component';
import { useClusterName } from 'hooks/api/useClusterName';

type DataSourceDeclaration = {
  provider: string;
  fieldPaths: string[];
};

const collectDataSources = (
  sections: Record<string, Section>
): DataSourceDeclaration[] => {
  const results: DataSourceDeclaration[] = [];
  const byKey = new Map<string, DataSourceDeclaration>();

  const walk = (components: Record<string, Component | ComponentGroup>) => {
    for (const comp of Object.values(components)) {
      if ('components' in comp) {
        walk((comp as ComponentGroup).components);
      }
      const asComponent = comp as Component;
      if (hasDataSource(asComponent)) {
        const fieldPath = getComponentSourcePath(asComponent);
        const existing = byKey.get(asComponent.dataSource.provider);
        if (existing) {
          if (fieldPath) existing.fieldPaths.push(fieldPath);
        } else {
          const decl: DataSourceDeclaration = {
            provider: asComponent.dataSource.provider,
            fieldPaths: fieldPath ? [fieldPath] : [],
          };
          byKey.set(asComponent.dataSource.provider, decl);
          results.push(decl);
        }
      }
    }
  };

  for (const section of Object.values(sections)) {
    if (section.components) {
      walk(section.components);
    }
  }

  return results;
};

const PrefetchItem = ({
  provider,
  namespace,
  fieldPaths,
}: {
  provider: string;
  namespace: string;
  fieldPaths: string[];
}) => {
  const cluster = useClusterName();
  const { options, isLoading } = useProviderOptions(provider, {
    namespace,
    cluster,
  });
  const { getValues, setValue } = useFormContext();

  useEffect(() => {
    if (isLoading) return;

    const validValues = options.map((o) => o.value);

    for (const path of fieldPaths) {
      const current = getValues(path);

      // this block checks if the current form value is not in the valid options list (can happen during namespace change)
      if (current && !validValues.includes(current as string)) {
        setValue(path, options.length > 0 ? options[0].value : '', {
          shouldValidate: true,
        });
      } else if (
        (current === '' || current === undefined || current === null) &&
        options.length > 0
      ) {
        setValue(path, options[0].value, { shouldValidate: true });
      }
    }
  }, [isLoading, options, fieldPaths, getValues, setValue]);

  return null;
};

type DataSourcePrefetcherProps = {
  sections: Record<string, Section>;
  namespace?: string;
};

export const DataSourcePrefetcher = ({
  sections,
  namespace,
}: DataSourcePrefetcherProps) => {
  const dataSources = useMemo(() => collectDataSources(sections), [sections]);

  if (!namespace) return null;

  // TODO: Support enable/disable toggle wrappers — when a component has
  // an on/off toggle (e.g. monitoring enabled/disabled), the prefetch
  // should only fire when the toggle is enabled. This will require
  // reading the toggle state from the form and conditionally including
  // the PrefetchItem.

  return (
    <>
      {dataSources.map((ds) => (
        <ComponentErrorBoundary
          key={ds.provider}
          componentName={`prefetch:${ds.provider}`}
        >
          <PrefetchItem
            provider={ds.provider}
            namespace={namespace}
            fieldPaths={ds.fieldPaths}
          />
        </ComponentErrorBoundary>
      ))}
    </>
  );
};
