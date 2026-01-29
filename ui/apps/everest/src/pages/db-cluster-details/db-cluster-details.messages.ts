export const Messages = {
  onDemandBackupModal: {
    subHead:
      'Create an immediate backup of this database, to store a snapshot of its current data.',
    backupName: 'Backup name',
    storageLocation: 'Backup storage',
    backupDetails: 'Backup details',
    headerMessage: 'Create backup now',
    submitMessage: 'Create',
  },
  schedulesBackupModal: {
    subHead:
      'Create a task that regularly backs up this database according to your specified schedule.',
  },
  noStorages: {
    alert:
      'You don’t have any backup storage set yet. Please set one before you proceed.',
    submitButton: 'Go to settings',
    cancel: 'Cancel',
  },
  backups: 'Backups',
  restores: 'Restores',
  overview: 'Overview',
  components: 'Components',
  logs: 'Logs',
  dbActions: 'Actions',
  restoringDb:
    'We are recovering your database. Do not perform any actions on the database until recovery is complete.',
  pitrError: `PITR can experience issues resulting from gaps, which may occur due to reasons such as disabling and then enabling PITR or technical issues like data loss.
    To ensure proper functioning of PITR, you need to take an additional full backup.
    `,
};
