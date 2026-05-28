// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: Re-enable when restore dialogs and PITR/PG limits are restored.
// import { DbEngineType } from '@percona/types';
// import { PG_SLOTS_LIMIT } from 'consts';

export const Messages = {
  deleteDialog: {
    header: 'Delete backup',
    content: (backupName: string) => (
      <>
        Are you sure you want to permanently delete <b>{backupName}</b> backup?
      </>
    ),
    // TODO: Re-enable when PITR/PG deletion logic is restored.
    // content: (
    //   backupName: string,
    //   dbType: DbEngineType,
    //   willDisablePITR: boolean
    // ) => (
    //   <>
    //     {dbType === DbEngineType.POSTGRESQL ? (
    //       <>
    //         Are you sure you want to permanently delete <b>{backupName}</b>{' '}
    //         backup? The backup data will not be deleted from the backup storage.
    //       </>
    //     ) : (
    //       <>
    //         Are you sure you want to permanently delete <b>{backupName}</b>{' '}
    //         backup?
    //       </>
    //     )}
    //     {willDisablePITR &&
    //       ' This will disable point-in-time recovery, as it requires a full backup.'}
    //   </>
    // ),
    alertMessage:
      'This action will permanently destroy your backup and you will not be able to recover it.',
    confirmButton: 'Delete',
    // TODO: Re-enable when PITR/PG deletion logic is restored.
    // checkboxMessage: 'Delete backups storage data',
  },
  // TODO: Re-enable when restore dialogs are restored.
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
  restore: 'Restore',
  restoreToNewDb: 'Create new DB',
  // pgMaximum: (slotsInUse: number) =>
  //   `Note: There is a maximum of 3 backup schedules for PostgreSQL. You are using ${slotsInUse} out of ${PG_SLOTS_LIMIT} available storages.`,
  // pitrToBeDisabled:
  //   'This will disable point-in-time recovery, as it requires a full backup.',
};
