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

import { useMemo } from 'react';
import {
  Alert,
  Box,
  Button,
  Link as MuiLink,
  Stack,
  Typography,
} from '@mui/material';
import { Link } from 'react-router-dom';
import { MRT_ColumnDef } from 'material-react-table';
import {
  NetworkNode as NetworkNodeIcon,
  OverviewCard,
  Table,
} from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';
import { DBClusterDetailsTabs } from 'pages/db-cluster-details/db-cluster-details.types';
import { flattenSchedules } from 'utils/backup-schedules';
import { BACKUP_STATUS_TO_BASE_STATUS } from 'pages/db-cluster-details/backups/backups-list/backups-list.constants';
import { getTimeSelectionPreviewMessage } from 'pages/database-form/database-preview/database.preview.messages';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { Backup, getBackupState } from 'shared-types/backups.types';
import OverviewSection from '../overview-section';
import OverviewSectionRow from '../overview-section-row';
import OverviewSectionText from '../overview-section-text/overview-section-text';
import { Messages } from '../cluster-overview.messages';
import { BackupsDetailsOverviewCardProps } from './card.types';
import { MAX_VISIBLE_BACKUPS } from './backups-details.constants';
import {
  formatBackupDate,
  getPitrEnabledStorageNames,
  getRecentBackups,
} from './backups-details.utils';

export const BackupsDetails = ({
  instance,
  namespace,
  loading,
}: BackupsDetailsOverviewCardProps) => {
  const instanceName = instance.metadata?.name ?? '';
  const clusterName = useClusterName();
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    {
      refetchInterval: 10 * 1000,
    }
  );

  const backupsPath = `/databases/${namespace}/${instanceName}/${DBClusterDetailsTabs.backups}`;

  const visibleBackups = useMemo(
    () => getRecentBackups(backups, MAX_VISIBLE_BACKUPS),
    [backups]
  );
  const hiddenBackupsCount = backups.length - visibleBackups.length;
  const backupsExist = backups.length > 0;
  const schedules = flattenSchedules(instance);
  const pitrStorages = getPitrEnabledStorageNames(instance);
  const pitrEnabled = pitrStorages.length > 0;

  const columns = useMemo<MRT_ColumnDef<Backup>[]>(
    () => [
      {
        accessorFn: getBackupState,
        id: 'state',
        header: '',
        maxSize: 20,
        Cell: ({ cell }) => (
          <StatusField
            status={cell.getValue<string>()}
            statusMap={BACKUP_STATUS_TO_BASE_STATUS}
          />
        ),
      },
      {
        accessorFn: (row) => row.status?.startedAt ?? '',
        id: 'startedAt',
        header: '',
        maxSize: 150,
        Cell: ({ cell }) => formatBackupDate(cell.getValue<string>()),
      },
      {
        accessorFn: (row) => row.status?.completedAt ?? '',
        id: 'completedAt',
        header: '',
        Cell: ({ cell }) => formatBackupDate(cell.getValue<string>()),
      },
    ],
    []
  );

  return (
    <OverviewCard
      dataTestId="backups-and-pitr"
      sx={{ width: '100%' }}
      cardHeaderProps={{
        title: Messages.titles.backups,
        avatar: <NetworkNodeIcon />,
        action: (
          <Button component={Link} size="small" to={backupsPath}>
            {Messages.actions.details}
          </Button>
        ),
      }}
    >
      <Stack
        sx={{
          gap: 3,
        }}
      >
        {backupsExist ? (
          <OverviewSection
            dataTestId="backups"
            title={
              <Box
                sx={{
                  display: 'flex',
                  flexDirection: 'row',
                  justifyContent: 'space-between',
                  width: '350px',
                }}
              >
                <Typography
                  color="text.primary"
                  variant="sectionHeading"
                  sx={{ marginLeft: '60px' }}
                >
                  {Messages.titles.started}
                </Typography>
                <Typography color="text.primary" variant="sectionHeading">
                  {Messages.titles.finished}
                </Typography>
              </Box>
            }
            loading={loading}
          >
            <Table
              getRowId={(row) => row.metadata?.name ?? ''}
              muiTopToolbarProps={{ sx: { display: 'none' } }}
              muiTableHeadCellProps={{ sx: { display: 'none' } }}
              enablePagination={false}
              enableBottomToolbar={false}
              tableName="backupList"
              data={visibleBackups}
              columns={columns}
            />
            {hiddenBackupsCount > 0 && (
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1 }}>
                <MuiLink
                  component={Link}
                  to={backupsPath}
                  variant="body2"
                  data-testid="see-other-backups"
                >
                  {Messages.actions.seeOtherBackups(hiddenBackupsCount)}
                </MuiLink>
              </Box>
            )}
          </OverviewSection>
        ) : (
          <Alert severity="info">{Messages.titles.noData}</Alert>
        )}
        <OverviewSection
          dataTestId="schedules"
          title={Messages.titles.schedules}
          loading={loading}
        >
          {schedules.length > 0 ? (
            schedules.map((schedule) => (
              <OverviewSectionText key={schedule.name}>
                {getTimeSelectionPreviewMessage(
                  getFormValuesFromCronExpression(schedule.cron)
                )}
              </OverviewSectionText>
            ))
          ) : (
            <OverviewSectionText>
              {Messages.fields.noSchedules}
            </OverviewSectionText>
          )}
        </OverviewSection>
        <OverviewSection
          dataTestId="pitr"
          title={Messages.titles.pitr}
          loading={loading}
        >
          <OverviewSectionRow
            dataTestId="pitr-status"
            labelProps={{ minWidth: '126px' }}
            label={Messages.fields.status}
            content={
              pitrEnabled ? Messages.fields.enabled : Messages.fields.disabled
            }
          />
          {pitrEnabled && (
            <OverviewSectionRow
              dataTestId="backup-storage"
              labelProps={{ minWidth: '126px' }}
              label={
                pitrStorages.length > 1
                  ? Messages.fields.backupStoragesPlural
                  : Messages.fields.backupStorages
              }
              content={
                <Stack>
                  {pitrStorages.map((storageName) => (
                    <Typography key={storageName} variant="body2">
                      {storageName}
                    </Typography>
                  ))}
                </Stack>
              }
            />
          )}
        </OverviewSection>
      </Stack>
    </OverviewCard>
  );
};
