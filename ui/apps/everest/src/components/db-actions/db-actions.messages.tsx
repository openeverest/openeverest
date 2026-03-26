export const Messages = {
  menuItems: {
    delete: 'Delete',
    restart: 'Restart',
    suspend: 'Suspend',
    resume: 'Resume',
    restoreFromBackup: 'Restore from a backup',
    createNewDbFromBackup: 'Create DB from a backup',
    dbStatusDetails: 'View DB status details',
    dbDetails: 'View details',
  },
  deleteModal: {
    header: 'Delete database',
    content: (dbName: string) => (
      <>
        Are you sure you want to permanently delete <b>{dbName}</b>? To confirm
        this action, type the name of your database.
      </>
    ),
    databaseName: 'Database name',
    alertMessage:
      'This action will permanently destroy your database and you will not be able to recover it.',
    checkboxMessage: 'Keep backups storage data',
    disabledCheckboxForPGTooltip:
      'Backups storage data is kept for PostgreSQL databases.',
    confirmButton: 'Delete',
  },
};
