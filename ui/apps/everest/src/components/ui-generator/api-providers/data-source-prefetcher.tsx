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
import { useFormContext, useWatch } from 'react-hook-form';
import type { Component, ComponentGroup, Section } from '../ui-generator.types';
import { hasDataSource } from './data-source-field';
import { useProviderOptions } from './registry';
import { ComponentErrorBoundary } from '../component-error-boundary';
import { getComponentSourcePath } from '../utils/preprocess/normalized-component';
import {
  TOGGLEABLE_SWITCHES_KEY,
  getToggleableMeta,
} from '../utils/toggleable/toggleable';
import { getByPath } from '../utils/object-path/object-path';
import { useClusterName } from 'hooks/api/useClusterName';
import { getReconciledDataSourceValue } from './data-source-field/data-source-field.utils';

type PrefetchField = {
  path?: string;
  // Switch of the toggleable group containing the field; fetched only when on.
  switchName?: string;
};

type DataSourceDeclaration = {
  provider: string;
  fields: PrefetchField[];
};

const collectDataSources = (
  sections: Record<string, Section>
): DataSourceDeclaration[] => {
  const results: DataSourceDeclaration[] = [];
  const byKey = new Map<string, DataSourceDeclaration>();

  const walk = (
    components: Record<string, Component | ComponentGroup>,
    switchName?: string
  ) => {
    for (const comp of Object.values(components)) {
      if ('components' in comp) {
        walk(
          comp.components,
          getToggleableMeta(comp)?.switchName ?? switchName
        );
        continue;
      }
      if (!hasDataSource(comp)) continue;

      const provider = comp.dataSource.provider;
      let decl = byKey.get(provider);
      if (!decl) {
        decl = { provider, fields: [] };
        byKey.set(provider, decl);
        results.push(decl);
      }
      decl.fields.push({ path: getComponentSourcePath(comp), switchName });
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

    for (const path of fieldPaths) {
      const nextValue = getReconciledDataSourceValue(getValues(path), options);

      if (nextValue !== null) {
        setValue(path, nextValue, { shouldValidate: true });
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
  const switches = useWatch({ name: TOGGLEABLE_SWITCHES_KEY });

  // A provider is mounted only while some of its fields are reachable, so a
  // switched-off section triggers no request.
  const reachable = useMemo(() => {
    const values = { [TOGGLEABLE_SWITCHES_KEY]: switches };
    return dataSources.flatMap(({ provider, fields }) => {
      const reachableFields = fields.filter(
        ({ switchName }) =>
          !switchName || getByPath(values, switchName) === true
      );
      if (reachableFields.length === 0) return [];
      const fieldPaths = reachableFields.flatMap(({ path }) =>
        path ? [path] : []
      );
      return [{ provider, fieldPaths }];
    });
  }, [dataSources, switches]);

  if (!namespace) return null;

  return (
    <>
      {reachable.map(({ provider, fieldPaths }) => (
        <ComponentErrorBoundary
          key={provider}
          componentName={`prefetch:${provider}`}
        >
          <PrefetchItem
            provider={provider}
            namespace={namespace}
            fieldPaths={fieldPaths}
          />
        </ComponentErrorBoundary>
      ))}
    </>
  );
};
