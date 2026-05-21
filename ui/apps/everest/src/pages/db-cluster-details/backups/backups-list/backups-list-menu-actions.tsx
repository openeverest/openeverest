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
// import AddIcon from '@mui/icons-material/Add';
// import KeyboardReturnIcon from '@mui/icons-material/KeyboardReturn';
import { MRT_Row } from 'material-react-table';
import { Backup } from 'shared-types/backups.types';
// import { BackupStatus } from 'shared-types/backups.types';
// import { DbCluster } from 'shared-types/dbCluster.types';
// import { useRBACPermissions } from 'hooks/rbac';
import { Messages } from './backups-list.messages';

// TODO: check main — original had restore/restoreToNewDb actions with RBAC checks.
// Restore these when restore feature is implemented in v2.
export const BackupActionButtons = (
  row: MRT_Row<Backup>,
  // TODO: v2 restore feature — uncomment when ready
  // blockActions: boolean,
  handleDeleteBackup: (backupName: string) => void
  // handleRestoreBackup: (backupName: string) => void,
  // handleRestoreToNewDbBackup: (backupName: string) => void,
  // dbCluster: DbCluster
) => {
  const backupName = row.original.metadata?.name ?? '';

  // TODO: v2 restore — original RBAC checks:
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
    // TODO: v2 restore — uncomment when restore feature is ready
    // ...(canRestore ? [
    //   <MenuItem key={0} disabled={row.original.state !== BackupStatus.OK || blockActions}
    //     onClick={() => handleRestoreBackup(row.original.name)}
    //     sx={{ m: 0, gap: 1, px: 2, py: '10px' }}>
    //     <KeyboardReturnIcon /> {Messages.restore}
    //   </MenuItem>,
    // ] : []),
    // ...(canCreateClusterFromBackup ? [
    //   <MenuItem key={1} disabled={row.original.state !== BackupStatus.OK || blockActions}
    //     onClick={() => handleRestoreToNewDbBackup(row.original.name)}
    //     sx={{ m: 0, gap: 1, px: 2, py: '10px' }}>
    //     <AddIcon /> {Messages.restoreToNewDb}
    //   </MenuItem>,
    // ] : []),
    <MenuItem
      key="delete"
      onClick={() => handleDeleteBackup(backupName)}
      sx={{ m: 0, gap: 1, px: 2, py: '10px' }}
    >
      <DeleteIcon />
      {Messages.delete}
    </MenuItem>,
  ];
};
