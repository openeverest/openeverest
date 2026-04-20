import { Backup } from 'shared-types/backupsOld.types';

export type BackupListTableHeaderProps = {
  onNowClick: () => void;
  onScheduleClick: () => void;
  noStoragesAvailable?: boolean;
  currentBackups: Backup[];
};
