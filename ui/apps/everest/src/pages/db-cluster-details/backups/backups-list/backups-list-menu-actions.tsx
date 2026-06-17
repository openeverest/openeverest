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
// import AddIcon from '@mui/icons-material/Add'; // TODO: re-enable when create-new-db is restored
import { MRT_Row } from 'material-react-table';
import {
  Backup,
  BackupStatus,
  getBackupState,
} from 'shared-types/backups.types';
// TODO: Re-enable when RBAC-based restore actions are restored.
// import { DbCluster } from 'shared-types/dbCluster.types';
import { Messages } from './backups-list.messages';

// TODO: check main — original had restore/restoreToNewDb actions with RBAC checks.
// Restore these when restore feature is implemented in v2.
export const getBackupActionButtons = (
  row: MRT_Row<Backup>,
  handleDeleteBackup: (backupName: string) => void,
  handleRestoreBackup: (backupName: string) => void,
  // handleRestoreToNewDbBackup: (backupName: string) => void, // TODO: re-enable when create-new-db is restored
  _handleRestoreToNewDbBackup: (backupName: string) => void,
  canDelete: boolean,
  isDeleting = false
) => {
  const backupName = row.original.metadata?.name ?? '';
  const backupState = getBackupState(row.original);
  // TODO: Re-enable when RBAC-based restore actions are restored.
  // const { canDelete } = useRBACPermissions(
  //   'database-cluster-backups',
  //   `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  // );
  // const { canCreate: canCreateRestore } = useRBACPermissions(
  //   'database-cluster-restores',
  //   `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  // );
  // const { canCreate: canCreateClusters } = useRBACPermissions(
  //   'database-clusters',
  //   `${dbCluster.metadata.namespace}/*`
  // );
  // const { canRead: canReadCredentials } = useRBACPermissions(
  //   'database-cluster-credentials',
  //   `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  // );
  // const canRestore = canCreateRestore && canReadCredentials;
  // const canCreateClusterFromBackup = canRestore && canCreateClusters;

  return [
    <MenuItem
      key="restore"
      disabled={backupState !== BackupStatus.SUCCEEDED}
      onClick={() => handleRestoreBackup(backupName)}
      sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    >
      <KeyboardReturnIcon /> {Messages.restore}
    </MenuItem>,
    // TODO: Temporarily hidden — create new DB from backup deferred by team
    // <MenuItem
    //   key="restore-to-new"
    //   disabled={backupState !== BackupStatus.SUCCEEDED}
    //   onClick={() => handleRestoreToNewDbBackup(backupName)}
    //   sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    // >
    //   <AddIcon /> {Messages.restoreToNewDb}
    // </MenuItem>,
    ...(canDelete
      ? [
          <MenuItem
            key="delete"
            disabled={backupState === BackupStatus.DELETING || isDeleting}
            onClick={() => handleDeleteBackup(backupName)}
            sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
          >
            <DeleteIcon />
            {Messages.delete}
          </MenuItem>,
        ]
      : []),
  ];
};
