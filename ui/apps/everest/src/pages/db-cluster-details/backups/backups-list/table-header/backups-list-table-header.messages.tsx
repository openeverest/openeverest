export const Messages = {
  now: 'Now',
  schedule: 'Schedule',
  deleteModal: {
    header: 'Delete schedule',
    content: (scheduleName: string, willDisablePITR: boolean) => (
      <>
        Are you sure you want to permanently delete schedule{' '}
        <b>{scheduleName}</b>?{' '}
        {willDisablePITR &&
          ' This will disable point-in-time recovery, as it requires a full backup.'}
      </>
    ),
  },
  activeSchedules: (schedulesNumber: number) =>
    `${schedulesNumber} active schedule${schedulesNumber > 1 ? 's' : ''}`,
  exceededScheduleBackupsNumber:
    'Maximum limit of schedules for PostgreSQL reached.',
  noStoragesAvailable:
    'Add a new backup storage in order to create a backup schedule',
};
