import { BaseStatus } from 'components/status-field/status-field.types';
import { BackupStatus } from 'shared-types/backups.types';

export const BACKUP_STATUS_TO_BASE_STATUS: Record<string, BaseStatus> = {
  [BackupStatus.PENDING]: 'pending',
  [BackupStatus.RUNNING]: 'pending',
  [BackupStatus.SUCCEEDED]: 'success',
  [BackupStatus.FAILED]: 'error',
  [BackupStatus.ERROR]: 'error',
  [BackupStatus.UNKNOWN]: 'unknown',
};
