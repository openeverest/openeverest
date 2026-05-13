export const Messages = {
  deleteDialog: {
    header: 'Delete backup',
    // TODO: Restore PG-specific delete message and PITR warning when v2 Instance types are finalized
    // content: (backupName: string, dbType: DbEngineType, willDisablePITR: boolean) => (...)
    content: (backupName: string) => (
      <>
        Are you sure you want to permanently delete <b>{backupName}</b> backup?
      </>
    ),
    alertMessage:
      'This action will permanently destroy your backup and you will not be able to recover it.',
    confirmButton: 'Delete',
    // TODO: Restore when cleanup backup storage feature is implemented for v2
    // checkboxMessage: 'Delete backups storage data',
  },
  // TODO: Restore when restore functionality is implemented for v2
  // restoreDialog: {
  //   header: 'Restore to this database',
  //   content:
  //     'Are you sure you want to restore the selected backup? This will update your database to the selected instance.',
  //   submitButton: 'Restore',
  // },
  // restoreDialogToNewDb: {
  //   header: 'Create database from backup',
  //   content:
  //     'Are you sure you want to replicate the selected database? This will create an exact copy of the current instance.',
  //   submitButton: 'Create',
  // },
  noData: 'You currently do not have any backups. Create one to get started.',
  createBackup: 'Create backup',
  now: 'Now',
  schedule: 'Schedule',
  delete: 'Delete',
  // TODO: Restore when restore functionality is implemented for v2
  // restore: 'Restore to this DB',
  // restoreToNewDb: 'Create new DB',
  // TODO: Restore when PG schedule limit is adapted for v2 Instance API
  // pgMaximum: (slotsInUse: number) =>
  //   `Note: There is a maximum of 3 backup schedules for PostgreSQL. You are using ${slotsInUse} out of ${PG_SLOTS_LIMIT} available storages.`,
  // pitrToBeDisabled:
  //   'This will disable point-in-time recovery, as it requires a full backup.',
};
