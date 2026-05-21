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

import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { useDbInstance } from 'hooks/api/db-instances/useDbInstance';
import { LastBackupProps } from './LastBackup.types';
import { Typography } from '@mui/material';
import {
  getLastBackupStatus,
  getLastBackupTimeDiff,
  sortBackupsByTime,
} from '../DbClusterView.utils';
import { BackupStatus } from 'shared-types/backups.types';
// TODO backups: check main — this file was migrated from v1 useDbBackups to v2 useBackupsList.
// Original v1 code used useDbBackups, useDbClusterPitr, useDbCluster, and had PITR gap warning.
// Verify nothing was lost when reviewing the backups table feature.

// Original v1 imports (removed because hooks no longer exist):
// import { useDbBackups, useDbClusterPitr } from 'hooks/api/backups/useBackups';
// import { IconButton, Tooltip, Typography } from '@mui/material';
// import { WarningIcon } from '@percona/ui-lib';
// import { useDbCluster } from 'hooks/api/db-cluster/useDbCluster';
// import { useNavigate } from 'react-router-dom';
// import { Messages } from '../dbClusterView.messages';

export const LastBackup = ({ dbName, namespace }: LastBackupProps) => {
  const clusterName = useClusterName();
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    dbName,
    {
      enabled: !!dbName,
      refetchInterval: 10_000,
    }
  );

  const { data: instance } = useDbInstance(namespace, dbName);

  const hasSchedules = (instance?.spec?.backup?.storages ?? []).some(
    (s) => s.schedules && s.schedules.length > 0
  );

  const finishedBackups = backups.filter(
    (backup) =>
      backup.status?.completedAt &&
      backup.status?.state === BackupStatus.SUCCEEDED
  );
  const sortedBackups = sortBackupsByTime(finishedBackups);
  const lastFinishedBackup = sortedBackups[sortedBackups.length - 1];
  const lastFinishedBackupDate = lastFinishedBackup?.status?.completedAt
    ? new Date(lastFinishedBackup.status.completedAt)
    : new Date();

  return (
    <>
      {finishedBackups.length ? (
        <>
          <Typography variant="body2">
            {getLastBackupTimeDiff(lastFinishedBackupDate)}
          </Typography>
          {/* TODO backups: wire PITR gaps warning when useDbClusterPitr is available */}
          {/* Original v1 PITR warning:
          {pitrData?.gaps && (
            <Tooltip
              title={Messages.lastBackup.warningTooltip}
              placement="right"
              arrow
            >
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(`${namespace}/${dbName}/backups`);
                }}
              >
                <WarningIcon />
              </IconButton>
            </Tooltip>
          )}
          */}
        </>
      ) : (
        <Typography variant="body2">
          {getLastBackupStatus(sortedBackups, hasSchedules)}
        </Typography>
      )}
    </>
  );
};
