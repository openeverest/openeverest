import { Backup } from 'shared-types/backups.types';

// TODO: Migrate scheduled backups list to v2 Instance API
// The original v2 component rendered a list of scheduled backups with edit/delete actions,
// RBAC checks, PITR logic, and PG-specific constraints.
// See git history (v2 branch) for full implementation.
const ScheduledBackupsList = (_props: { currentBackups: Backup[] }) => {
  return null;
};

export default ScheduledBackupsList;
