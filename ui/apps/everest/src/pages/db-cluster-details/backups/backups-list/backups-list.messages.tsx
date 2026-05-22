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
