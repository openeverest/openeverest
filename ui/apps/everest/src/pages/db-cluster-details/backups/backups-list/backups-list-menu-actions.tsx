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
// import AddIcon from '@mui/icons-material/Add';
// import KeyboardReturnIcon from '@mui/icons-material/KeyboardReturn';
import DeleteIcon from '@mui/icons-material/Delete';
import { MRT_Row } from 'material-react-table';
import { Backup, BackupStatus } from 'shared-types/backups.types';
import { Messages } from './backups-list.messages';
// import { useRBACPermissions } from 'hooks/rbac';
// import { Instance } from 'shared-types/api.types';

export const BackupActionButtons = (
  row: MRT_Row<Backup>,
  // TODO: Restore when restore functionality is implemented for v2
  // blockActions: boolean,
  handleDeleteBackup: (backupName: string) => void
  // handleRestoreBackup: (backupName: string) => void,
  // handleRestoreToNewDbBackup: (backupName: string) => void,
  // instance: Instance
) => {
  const backupName = row.original.metadata?.name ?? '';
  const backupState = row.original.status?.state;

  // TODO: Restore RBAC checks when v2 RBAC resource names are finalized
  // const { canDelete } = useRBACPermissions(
  //   'backups',
  //   `${instance.metadata?.namespace}/${backupName}`
  // );
  // const { canCreate: canCreateRestore } = useRBACPermissions(
  //   'restores',
  //   `${instance.metadata?.namespace}/${backupName}`
  // );
  // const { canCreate: canCreateInstances } = useRBACPermissions(
  //   'instances',
  //   `${instance.metadata?.namespace}/*`
  // );
  // const canRestore = canCreateRestore;
  // const canCreateInstanceFromBackup = canRestore && canCreateInstances;

  return [
    // TODO: Restore when restore functionality is implemented for v2
    // ...(canRestore
    //   ? [
    //       <MenuItem
    //         key={0}
    //         disabled={backupState !== BackupStatus.SUCCEEDED || blockActions}
    //         onClick={() => handleRestoreBackup(backupName)}
    //         sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    //       >
    //         <KeyboardReturnIcon />
    //         {Messages.restore}
    //       </MenuItem>,
    //     ]
    //   : []),
    // ...(canCreateInstanceFromBackup
    //   ? [
    //       <MenuItem
    //         key={1}
    //         disabled={backupState !== BackupStatus.SUCCEEDED || blockActions}
    //         onClick={() => handleRestoreToNewDbBackup(backupName)}
    //         sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    //       >
    //         <AddIcon />
    //         {Messages.restoreToNewDb}
    //       </MenuItem>,
    //     ]
    //   : []),
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
