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
import type {
  Component,
  ComponentGroup,
  DataSourceConfig,
  Section,
} from '../ui-generator.types';
import { hasDataSource } from './data-source-field';
import { useProviderOptions } from './registry';
import { ComponentErrorBoundary } from '../component-error-boundary';
import { getComponentSourcePath } from '../utils/preprocess/normalized-component';

type DataSourceDeclaration = {
  provider: string;
  config?: DataSourceConfig;
  fieldPaths: string[];
};

const collectDataSources = (
  sections: Record<string, Section>
): DataSourceDeclaration[] => {
  const results: DataSourceDeclaration[] = [];
  const byProvider = new Map<string, DataSourceDeclaration>();

  const walk = (components: Record<string, Component | ComponentGroup>) => {
    for (const comp of Object.values(components)) {
      if ('components' in comp) {
        walk((comp as ComponentGroup).components);
      }
      const asComponent = comp as Component;
      if (hasDataSource(asComponent)) {
        const fieldPath = getComponentSourcePath(asComponent);
        const existing = byProvider.get(asComponent.dataSource.provider);
        if (existing) {
          if (fieldPath) existing.fieldPaths.push(fieldPath);
        } else {
          const decl: DataSourceDeclaration = {
            provider: asComponent.dataSource.provider,
            config: asComponent.dataSource.config,
            fieldPaths: fieldPath ? [fieldPath] : [],
          };
          byProvider.set(asComponent.dataSource.provider, decl);
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
  cluster,
  config,
  fieldPaths,
}: {
  provider: string;
  namespace: string;
  cluster: string;
  config?: DataSourceConfig;
  fieldPaths: string[];
}) => {
  const { options, isLoading } = useProviderOptions(provider, {
    namespace,
    cluster,
    config,
  });
  const { getValues, setValue } = useFormContext();

  useEffect(() => {
    if (isLoading || options.length === 0) return;

    for (const path of fieldPaths) {
      const current = getValues(path);
      if (current === '' || current === undefined || current === null) {
        setValue(path, options[0].value, { shouldValidate: true });
      }
    }
  }, [isLoading, options, fieldPaths, getValues, setValue]);

  return null;
};

type DataSourcePrefetcherProps = {
  sections: Record<string, Section>;
  namespace?: string;
  cluster?: string;
};

export const DataSourcePrefetcher = ({
  sections,
  namespace,
  cluster,
}: DataSourcePrefetcherProps) => {
  const dataSources = useMemo(() => collectDataSources(sections), [sections]);

  if (!namespace || !cluster) return null;

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
            cluster={cluster}
            config={ds.config}
            fieldPaths={ds.fieldPaths}
          />
        </ComponentErrorBoundary>
      ))}
    </>
  );
};
