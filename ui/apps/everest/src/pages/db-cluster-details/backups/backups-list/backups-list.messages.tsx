export const Messages = {
  deleteDialog: {
    header: 'Delete backup',
    content: (backupName: string) => (
      <>
        Are you sure you want to permanently delete <b>{backupName}</b> backup?
      </>
    ),
    alertMessage:
      'This action will permanently destroy your backup and you will not be able to recover it.',
    confirmButton: 'Delete',
  },
  // TODO: check main — original had restore dialog messages with dbType/PITR logic:
  // restoreDialog: {
  //   header: 'Restore from a backup',
  //   content: (dbType: DbType, willDisablePITR: boolean) => ( ... ),
  //   checkboxMessage: 'This will also disable point-in-time recovery',
  // },
  // restoreDialogToNewDb: {
  //   header: 'Create a database from backup',
  // },
  // restore: 'Restore to this DB',
  // restoreToNewDb: 'Create new DB from backup',
  // pgMaximum: 'PG clusters support a maximum of 3 backup storages',
  // pitrToBeDisabled: 'Restore to this DB will disable PITR',
  noData: 'You currently do not have any backups. Create one to get started.',
  createBackup: 'Create backup',
  now: 'Now',
  schedule: 'Schedule',
  delete: 'Delete',
};
