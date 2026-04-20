// everest
// Copyright (C) 2023 Percona LLC
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

import { MRT_Row } from 'material-react-table';
import { MenuItem } from '@mui/material';
import { Messages } from './backups-list.messages';
import AddIcon from '@mui/icons-material/Add';
import KeyboardReturnIcon from '@mui/icons-material/KeyboardReturn';
import DeleteIcon from '@mui/icons-material/Delete';
import { DbCluster } from 'shared-types/dbCluster.types';
import { Backup, BackupStatus } from 'shared-types/backupsOld.types';
import { useRBACPermissions } from 'hooks/rbac';

export const BackupActionButtons = (
  row: MRT_Row<Backup>,
  // Only applies to restores, not to deletes
  blockActions: boolean,
  handleDeleteBackup: (backupName: string) => void,
  handleRestoreBackup: (backupName: string) => void,
  handleRestoreToNewDbBackup: (backupName: string) => void,
  dbCluster: DbCluster
) => {
  const { canDelete } = useRBACPermissions(
    'database-cluster-backups',
    `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  );
  const { canCreate: canCreateRestore } = useRBACPermissions(
    'database-cluster-restores',
    `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  );
  const { canCreate: canCreateClusters } = useRBACPermissions(
    'database-clusters',
    `${dbCluster.metadata.namespace}/*`
  );
  const { canRead: canReadCredentials } = useRBACPermissions(
    'database-cluster-credentials',
    `${dbCluster.metadata.namespace}/${row.original.dbClusterName}`
  );

  const canRestore = canCreateRestore && canReadCredentials;
  const canCreateClusterFromBackup = canRestore && canCreateClusters;

  return [
    ...(canRestore
      ? [
          <MenuItem
            key={0}
            disabled={row.original.state !== BackupStatus.OK || blockActions}
            onClick={() => {
              handleRestoreBackup(row.original.name);
            }}
            sx={{
              m: 0,
              gap: 1,
              px: 2,
              py: '10px',
            }}
          >
            <KeyboardReturnIcon />
            {Messages.restore}
          </MenuItem>,
        ]
      : []),
    ...(canCreateClusterFromBackup
      ? [
          <MenuItem
            key={1}
            disabled={row.original.state !== BackupStatus.OK || blockActions}
            onClick={() => {
              handleRestoreToNewDbBackup(row.original.name);
            }}
            sx={{
              m: 0,
              gap: 1,
              px: 2,
              py: '10px',
            }}
          >
            <AddIcon />
            {Messages.restoreToNewDb}
          </MenuItem>,
        ]
      : []),
    ...(canDelete
      ? [
          <MenuItem
            key={2}
            onClick={() => {
              handleDeleteBackup(row.original.name);
            }}
            disabled={row.original.state === BackupStatus.DELETING}
            sx={{
              m: 0,
              gap: 1,
              px: 2,
              py: '10px',
            }}
          >
            <DeleteIcon />
            {Messages.delete}
          </MenuItem>,
        ]
      : []),
  ];
};
