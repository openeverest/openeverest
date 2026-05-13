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
import { MRT_Row } from 'material-react-table';
import { Backup, BackupStatus } from 'shared-types/backups.types';
import { Messages } from './backups-list.messages';

export const BackupActionButtons = (
  row: MRT_Row<Backup>,
  handleDeleteBackup: (backupName: string) => void
) => {
  const backupName = row.original.metadata?.name ?? '';
  const backupState = row.original.status?.state;

  return [
    <MenuItem
      key="delete"
      onClick={() => handleDeleteBackup(backupName)}
      disabled={backupState === BackupStatus.PENDING}
      sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    >
      <DeleteIcon />
      {Messages.delete}
    </MenuItem>,
  ];
};
