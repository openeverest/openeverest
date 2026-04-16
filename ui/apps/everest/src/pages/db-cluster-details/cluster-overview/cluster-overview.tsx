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

import { useState, useCallback } from 'react';
import { Box, Stack } from '@mui/material';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';
import { Messages } from './cluster-overview.messages';
import { useClusterOverviewData } from './hooks/use-cluster-overview-data';
import BasicInfoSection from './sections/basic-info-section';
import ConnectionSection from './sections/connection-section';
import SchemaDrivenCard from './sections/schema-driven-card';
import OtherFieldsCard from './sections/other-fields-card';
import { SectionEditModal } from './sections/section-edit-modal';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { isSectionEditable } from 'components/ui-generator/utils/section-editable';
import { shouldDbActionsBeBlocked } from 'utils/db';

export const ClusterOverview = () => {
  const {
    namespace,
    instance,
    isLoading,
    credentials,
    schemaSectionCards,
    otherFields,
    provider,
    sections,
  } = useClusterOverviewData();

  const [editingSectionKey, setEditingSectionKey] = useState<string | null>(
    null
  );

  const handleCloseModal = useCallback(() => setEditingSectionKey(null), []);

  if (isLoading || !instance) {
    return null;
  }

  const actionsBlocked = shouldDbActionsBeBlocked(instance?.status?.phase);

  return (
    <Box
      sx={{
        columnCount: { xs: 1, lg: 2, xl: 3 },
        columnGap: 2,
        '& > *': { breakInside: 'avoid', marginBottom: 2 },
      }}
      data-testid="cluster-overview"
    >
      <Box>
        <OverviewCard
          dataTestId="database-details"
          sx={{ width: '100%' }}
          cardHeaderProps={{
            title: Messages.titles.dbDetails,
            avatar: <DatabaseIcon />,
          }}
        >
          <Stack gap={3}>
            <BasicInfoSection
              instance={instance}
              namespace={namespace}
              loading={isLoading}
            />
            <ConnectionSection credentials={credentials} loading={isLoading} />
          </Stack>
        </OverviewCard>
      </Box>
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
      {otherFields.length > 0 && (
        <OtherFieldsCard fields={otherFields} loading={isLoading} />
      )}

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

      {/* TODO: BackupsDetails card — re-enable once connected to new instance API */}
    </Box>
  );
};
