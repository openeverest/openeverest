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
