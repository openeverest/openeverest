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

import { MenuItem } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import KeyboardReturnIcon from '@mui/icons-material/KeyboardReturn';
import AddIcon from '@mui/icons-material/Add';
import { MRT_Row } from 'material-react-table';
import {
  Backup,
  BackupStatus,
  getBackupState,
} from 'shared-types/backups.types';
import { Messages } from './backups-list.messages';

interface BackupActionButtonsParams {
  row: MRT_Row<Backup>;
  handlers: {
    onRestore: (backupName: string) => void;
    onRestoreToNewDb: (backupName: string) => void;
    onDelete: (backupName: string) => void;
  };
  permissions: {
    canRestore: boolean;
    canCreateNewDb: boolean;
    canDelete: boolean;
  };
  isDeleting?: boolean;
}

// Row-level backup actions: restore in place, clone into a new DB, and delete.
// Each item is gated by the RBAC flags the caller resolves.
export const getBackupActionButtons = ({
  row,
  handlers,
  permissions,
  isDeleting = false,
}: BackupActionButtonsParams) => {
  const backupName = row.original.metadata?.name ?? '';
  const backupState = getBackupState(row.original);

  return [
    ...(permissions.canRestore
      ? [
          <MenuItem
            key="restore"
            disabled={backupState !== BackupStatus.SUCCEEDED}
            onClick={() => handlers.onRestore(backupName)}
            sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
          >
            <KeyboardReturnIcon /> {Messages.restore}
          </MenuItem>,
        ]
      : []),
    ...(permissions.canCreateNewDb
      ? [
          <MenuItem
            key="restore-to-new"
            disabled={backupState !== BackupStatus.SUCCEEDED}
            onClick={() => handlers.onRestoreToNewDb(backupName)}
            sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
          >
            <AddIcon /> {Messages.restoreToNewDb}
          </MenuItem>,
        ]
      : []),
    ...(permissions.canDelete
      ? [
          <MenuItem
            key="delete"
            disabled={backupState === BackupStatus.DELETING || isDeleting}
            onClick={() => handlers.onDelete(backupName)}
            sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
          >
            <DeleteIcon />
            {Messages.delete}
          </MenuItem>,
        ]
      : []),
  ];
};
