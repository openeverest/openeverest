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

import { useContext, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { DbInstanceContext } from '../../dbCluster.context';
import { useDbInstanceCredentials } from 'hooks/api/db-instances/useCreateDbInstance';
import { useProviders } from 'hooks/api/providers';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import type {
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import type { Provider } from 'shared-types/api.types';
import { collectSectionFields } from '../utils/cluster-overview.helpers';
import type { SectionField } from '../utils/cluster-overview.helpers';
import {
  flattenObject,
  formatDisplayValue,
} from 'components/ui-generator/utils/object-path/object-path';
import { collectAllSchemaPaths } from 'components/ui-generator/utils/schema-walker';

export interface SchemaSectionCard {
  key: string;
  title: string;
  fields: SectionField[];
}

export interface UncoveredField {
  label: string;
  value: string;
}

export const useClusterOverviewData = () => {
  const { instanceName, namespace = '' } = useParams();
  const { instance, isLoading, canReadCredentials } =
    useContext(DbInstanceContext);

  const { data: credentials } = useDbInstanceCredentials(
    instanceName || '',
    namespace,
    {
      enabled:
        canReadCredentials &&
        !!instanceName &&
        instance?.status?.phase === 'Ready',
    }
  );

  const { data: providers } = useProviders();

  const provider: Provider | undefined = useMemo(
    () =>
      instance?.spec?.provider && providers
        ? providers.find((p) => p.metadata?.name === instance.spec.provider)
        : undefined,
    [instance?.spec?.provider, providers]
  );

  const { sections, sectionsOrder } = useMemo(() => {
    const empty = {
      sections: {} as Record<string, Section>,
      sectionsOrder: [] as string[],
    };
    if (!provider?.spec?.uiSchema) return empty;

    const schema = preprocessSchema(
      provider.spec.uiSchema as TopologyUISchemas,
      provider
    );
    const topologyType = instance?.spec?.topology?.type;
    const topology = topologyType
      ? schema[topologyType]
      : Object.values(schema)[0];

    if (!topology) return empty;

    return {
      sections: topology.sections,
      sectionsOrder: topology.sectionsOrder ?? Object.keys(topology.sections),
    };
  }, [provider, instance?.spec?.topology?.type]);

  // Schema-driven section cards
  const instanceAsRecord = instance as unknown as Record<string, unknown>;

  const schemaSectionCards: SchemaSectionCard[] = useMemo(() => {
    if (!instance || !sections) return [];

    return sectionsOrder
      .map((sectionKey) => {
        const section = sections[sectionKey];
        if (!section) return null;

        return {
          key: sectionKey,
          title: section.label ?? sectionKey,
          fields: collectSectionFields(
            section.components,
            instanceAsRecord,
            section.componentsOrder
          ),
        };
      })
      .filter(Boolean) as SchemaSectionCard[];
  }, [instance, sections, sectionsOrder, instanceAsRecord]);

  // TODO add more tests
  // Uncovered fields (not in schema)
  const otherFields: UncoveredField[] = useMemo(() => {
    if (!instance) return [];

    const coveredPaths = collectAllSchemaPaths(sections);
    coveredPaths.add('metadata.name');
    coveredPaths.add('spec.provider');
    coveredPaths.add('spec.topology.type');

    return flattenObject(instance.spec, 'spec')
      .filter(({ key }) => !coveredPaths.has(key))
      .map(({ key, value }) => ({
        label: key,
        value: formatDisplayValue(value),
      }));
  }, [instance, sections]);

  return {
    instanceName,
    namespace,
    instance,
    isLoading,
    credentials,
    schemaSectionCards,
    otherFields,
    provider,
    sections,
    selectedTopology: instance?.spec?.topology?.type,
  };
};
