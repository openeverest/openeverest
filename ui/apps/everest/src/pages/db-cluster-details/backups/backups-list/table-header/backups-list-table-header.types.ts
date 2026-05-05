import { Backup } from 'shared-types/backups.types';

export type BackupListTableHeaderProps = {
  onNowClick: () => void;
  onScheduleClick: () => void;
  noStoragesAvailable?: boolean;
  currentBackups: Backup[];
};
