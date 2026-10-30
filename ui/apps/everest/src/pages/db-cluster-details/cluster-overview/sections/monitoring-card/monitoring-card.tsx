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

import { useMemo, useState } from 'react';
import { Stack } from '@mui/material';
import { EditIconButton, OverviewCard } from '@percona/ui-lib';
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { useRBACPermissions } from 'hooks/rbac';
import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import OverviewSection from '../../overview-section';
import OverviewSectionRow from '../../overview-section-row';
import { MonitoringEditModal } from '../monitoring-edit-modal/monitoring-edit-modal';
import { ConfiguredStatus } from './configured-status/configured-status';
import { Messages } from './monitoring-card.messages';
import { MonitoringCardProps } from './monitoring-card.types';
import { getMonitoringOverview } from './monitoring-card.utils';

const NO_CLASSES: MonitoringClass[] = [];

// Shown when the provider can be monitored or the Instance already is.
export const MonitoringCard = ({
  instance,
  namespace,
  editable,
}: MonitoringCardProps) => {
  const [editing, setEditing] = useState(false);
  const { data: classes = NO_CLASSES } = useMonitoringClassesForProvider(
    instance.spec?.providerRef.name ?? ''
  );
  const { canUpdate } = useRBACPermissions(
    'instances',
    `${namespace}/${instance.metadata?.name ?? ''}`
  );
  const { configured, entries } = useMemo(
    () => getMonitoringOverview(instance),
    [instance]
  );

  if (classes.length === 0 && entries.length === 0) {
    return null;
  }

  return (
    <>
      <OverviewCard
        dataTestId="monitoring-details"
        cardHeaderProps={{
          title: Messages.title,
          action:
            editable && canUpdate && classes.length > 0 ? (
              <EditIconButton
                dataTestId="monitoring-edit-button"
                onClick={() => setEditing(true)}
                label={Messages.edit}
              />
            ) : undefined,
        }}
      >
        <Stack sx={{ gap: 3 }}>
          {entries.length === 0 ? (
            <OverviewSection dataTestId="monitoring">
              <OverviewSectionRow
                label={Messages.destinations}
                content={Messages.notConfigured}
              />
            </OverviewSection>
          ) : (
            <>
              {configured && (
                <OverviewSection dataTestId="monitoring-configured">
                  <OverviewSectionRow
                    label={Messages.status}
                    content={<ConfiguredStatus status={configured.status} />}
                  />
                  {configured.details && (
                    <OverviewSectionRow
                      label={Messages.details}
                      content={configured.details}
                    />
                  )}
                </OverviewSection>
              )}
              {entries.map((entry) => (
                <OverviewSection
                  key={entry.name}
                  title={entry.destination}
                  dataTestId={`monitoring-${entry.name}`}
                >
                  <OverviewSectionRow
                    label={Messages.status}
                    content={<ConfiguredStatus status={entry.configured} />}
                  />
                  {entry.mode && (
                    <OverviewSectionRow
                      label={Messages.mode}
                      content={entry.mode}
                    />
                  )}
                  {entry.details && (
                    <OverviewSectionRow
                      label={Messages.details}
                      content={entry.details}
                    />
                  )}
                </OverviewSection>
              ))}
            </>
          )}
        </Stack>
      </OverviewCard>
      {editing && (
        <MonitoringEditModal
          instance={instance}
          namespace={namespace}
          onClose={() => setEditing(false)}
        />
      )}
    </>
  );
};
