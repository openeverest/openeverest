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

import { Box, Stack } from '@mui/material';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';
import { InstanceConnectionDetails } from 'types/api';
import OverviewSection from './overview-section';
import OverviewSectionRow from './overview-section-row';
import { Messages } from './cluster-overview.messages';
import { useClusterOverviewData } from './hooks/use-cluster-overview-data';
import type {
  SchemaSectionCard,
  UncoveredField,
} from './hooks/use-cluster-overview-data';
import type { Instance } from 'types/api';

const BasicInfoSection = ({
  instance,
  namespace,
  loading,
}: {
  instance: Instance;
  namespace: string;
  loading: boolean;
}) => (
  <OverviewSection
    dataTestId="basic-information"
    title={Messages.titles.basicInformation}
    loading={loading}
  >
    <OverviewSectionRow
      label={Messages.fields.name}
      content={instance.metadata?.name ?? '\u2014'}
    />
    <OverviewSectionRow
      label={Messages.fields.namespace}
      content={namespace || '\u2014'}
    />
    <OverviewSectionRow
      label="Provider"
      content={instance.spec?.provider ?? '\u2014'}
    />
    <OverviewSectionRow
      label="Topology"
      content={instance.spec?.topology?.type ?? 'default'}
    />
    <OverviewSectionRow
      label={Messages.fields.status}
      content={instance.status?.phase ?? '\u2014'}
    />
  </OverviewSection>
);

const ConnectionSection = ({
  credentials,
  loading,
}: {
  credentials: InstanceConnectionDetails | undefined;
  loading: boolean;
}) => (
  <OverviewSection
    dataTestId="connection-details"
    title={Messages.titles.connectionDetails}
    loading={loading}
  >
    {credentials ? (
      <>
        <OverviewSectionRow
          label={Messages.fields.host}
          content={credentials.host ?? '\u2014'}
        />
        <OverviewSectionRow
          label={Messages.fields.port}
          content={String(credentials.port ?? '\u2014')}
        />
        <OverviewSectionRow
          label={Messages.fields.username}
          content={credentials.username ?? '\u2014'}
        />
        <OverviewSectionRow
          label={Messages.fields.connectionUrl}
          content={credentials.uri ?? '\u2014'}
        />
      </>
    ) : (
      <OverviewSectionRow
        label={Messages.fields.status}
        content="Waiting for instance to be ready..."
      />
    )}
  </OverviewSection>
);

const SchemaDrivenCard = ({
  card,
  loading,
}: {
  card: SchemaSectionCard;
  loading: boolean;
}) => (
  <Box>
    <OverviewCard
      dataTestId={`${card.key}-details`}
      sx={{ width: '100%' }}
      cardHeaderProps={{
        title: card.title,
        avatar: <DatabaseIcon />,
      }}
    >
      <Stack gap={3}>
        <OverviewSection dataTestId={card.key} loading={loading}>
          {card.fields.length > 0 ? (
            card.fields.map(({ label, value }) => (
              <OverviewSectionRow key={label} label={label} content={value} />
            ))
          ) : (
            <OverviewSectionRow label="Info" content="No data available" />
          )}
        </OverviewSection>
      </Stack>
    </OverviewCard>
  </Box>
);

const OtherFieldsCard = ({
  fields,
  loading,
}: {
  fields: UncoveredField[];
  loading: boolean;
}) => (
  <Box>
    <OverviewCard
      dataTestId="other-details"
      sx={{ width: '100%' }}
      cardHeaderProps={{
        title: 'Other',
        avatar: <DatabaseIcon />,
      }}
    >
      <Stack gap={3}>
        <OverviewSection dataTestId="other" loading={loading}>
          {fields.map(({ label, value }) => (
            <OverviewSectionRow key={label} label={label} content={value} />
          ))}
        </OverviewSection>
      </Stack>
    </OverviewCard>
  </Box>
);

export const ClusterOverview = () => {
  const {
    namespace,
    instance,
    isLoading,
    credentials,
    schemaSectionCards,
    otherFields,
  } = useClusterOverviewData();

  if (isLoading || !instance) {
    return null;
  }

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
            <ConnectionSection
              credentials={credentials}
              loading={isLoading}
            />
          </Stack>
        </OverviewCard>
      </Box>
      {schemaSectionCards.map((card) => (
        <SchemaDrivenCard key={card.key} card={card} loading={isLoading} />
      ))}

      {/* Uncovered instance fields */}
      {otherFields.length > 0 && (
        <OtherFieldsCard fields={otherFields} loading={isLoading} />
      )}

      {/* TODO: BackupsDetails card — re-enable once connected to new instance API */}
    </Box>
  );
};
