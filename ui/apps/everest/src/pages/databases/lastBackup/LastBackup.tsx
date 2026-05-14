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
          {/* TODO: wire PITR gaps warning when useDbClusterPitr is available */}
        </>
      ) : (
        <Typography variant="body2">
          {getLastBackupStatus(sortedBackups, hasSchedules)}
        </Typography>
      )}
    </>
  );
};
