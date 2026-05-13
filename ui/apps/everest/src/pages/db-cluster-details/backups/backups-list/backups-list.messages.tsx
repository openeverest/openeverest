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
  noData: 'You currently do not have any backups. Create one to get started.',
  createBackup: 'Create backup',
  now: 'Now',
  schedule: 'Schedule',
  delete: 'Delete',
};
