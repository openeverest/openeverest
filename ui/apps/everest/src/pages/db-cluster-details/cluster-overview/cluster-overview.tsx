// everest
// Copyright (C) 2023 Percona LLC
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

import { useMemo, useState, useCallback } from 'react';
import { Stack } from '@mui/material';
import { DatabaseIcon, MasonryColumns, OverviewCard } from '@percona/ui-lib';
import { Messages } from './cluster-overview.messages';
import { useClusterOverviewData } from './hooks/use-cluster-overview-data';
import BasicInfoSection from './sections/basic-info-section';
import ConnectionSection from './sections/connection-section';
import SchemaDrivenCard from './sections/schema-driven-card';
import { BackupsDetails } from './old-cards/backups-details';
// TODO: Re-enable OtherFieldsCard when uncovered fields are properly formatted
// import OtherFieldsCard from './sections/other-fields-card';
import { SectionEditModal } from './sections/section-edit-modal';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { isSectionEditable } from 'components/ui-generator/utils/section-editable';
import { shouldDbActionsBeBlocked } from 'utils/db';
import { usePlugins } from 'contexts/plugins';
import type { ClusterCardExtension } from '@openeverest/plugin-sdk';
import PluginErrorBoundary from 'components/plugin-host/PluginErrorBoundary';

export const ClusterOverview = () => {
  const {
    namespace,
    instance,
    isLoading,
    credentials,
    schemaSectionCards,
    // otherFields, // TODO: Re-enable when OtherFieldsCard is restored
    provider,
    sections,
    backupsSupported,
  } = useClusterOverviewData();

  const [editingSectionKey, setEditingSectionKey] = useState<string | null>(
    null
  );

  const handleCloseModal = useCallback(() => setEditingSectionKey(null), []);

  // Collect plugin clusterCard extensions.
  const { plugins } = usePlugins();
  const pluginCards = useMemo(
    () =>
      plugins.flatMap((p) =>
        p.extensions
          .filter(
            (ext): ext is ClusterCardExtension => ext.type === 'clusterCard'
          )
          .map((ext) => ({ pluginName: p.name, ext }))
      ),
    [plugins]
  );

  if (isLoading || !instance) {
    return null;
  }

  const actionsBlocked = shouldDbActionsBeBlocked(instance?.status?.phase);

  return (
    <>
      <MasonryColumns
        minColumnWidth={440}
        maxColumns={3}
        dataTestId="cluster-overview"
      >
        <OverviewCard
          dataTestId="database-details"
          cardHeaderProps={{
            title: Messages.titles.dbDetails,
            avatar: <DatabaseIcon />,
          }}
        >
          <Stack
            sx={{
              gap: 3,
            }}
          >
            <BasicInfoSection
              instance={instance}
              namespace={namespace}
              loading={isLoading}
            />
            <ConnectionSection credentials={credentials} loading={isLoading} />
          </Stack>
        </OverviewCard>
        {schemaSectionCards.map((card) => {
          const section = sections[card.key];
          const editable =
            !actionsBlocked &&
            !!section &&
            isSectionEditable(section, FormMode.Edit);
          return (
            <SchemaDrivenCard
              key={card.key}
              card={card}
              loading={isLoading}
              editable={editable}
              onEdit={() => setEditingSectionKey(card.key)}
            />
          );
        })}
        {/* Uncovered instance fields */}
        {/* TODO: temporarily hidden until properly formatted
        {otherFields.length > 0 && (
          <OtherFieldsCard fields={otherFields} loading={isLoading} />
        )}
        */}
        {/* Plugin-contributed cards */}
        {pluginCards.map((pc) => {
          const CardComponent = pc.ext.component;
          return (
            <OverviewCard
              key={`plugin-card-${pc.pluginName}-${pc.ext.label}`}
              dataTestId={`plugin-card-${pc.pluginName}`}
              cardHeaderProps={{ title: pc.ext.label }}
            >
              <PluginErrorBoundary pluginName={pc.pluginName}>
                <CardComponent cluster={instance} namespace={namespace} />
              </PluginErrorBoundary>
            </OverviewCard>
          );
        })}
        {backupsSupported && (
          <BackupsDetails
            instance={instance}
            namespace={namespace}
            loading={isLoading}
          />
        )}
      </MasonryColumns>
      {editingSectionKey && provider && (
        <SectionEditModal
          sectionKey={editingSectionKey}
          sections={sections}
          instance={instance}
          provider={provider}
          namespace={namespace}
          onClose={handleCloseModal}
          onSuccess={handleCloseModal}
        />
      )}
    </>
  );
};
