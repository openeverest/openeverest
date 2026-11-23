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
import { Delete, Edit } from '@mui/icons-material';
import TableActionsMenu from 'components/table-actions-menu';
import { useRBACPermissions } from 'hooks/rbac';
import { BackupStorageState } from 'shared-types/backupStorages.types';
import { Messages } from '../storage-locations.messages';
import { BackupStorageRow } from '../storage-locations.types';

interface StorageRowActionsProps {
  row: BackupStorageRow;
  onEdit: (row: BackupStorageRow) => void;
  onDelete: (row: BackupStorageRow) => void;
}

// Permissions are checked per storage: a policy may grant them on some
// storages of a namespace only.
export const StorageRowActions = ({
  row,
  onEdit,
  onDelete,
}: StorageRowActionsProps) => {
  const { canUpdate, canDelete } = useRBACPermissions(
    'backup-storages',
    `${row.namespace}/${row.name}`
  );
  // The edit form is built from the storage's class.
  const { canRead: canReadClasses } = useRBACPermissions(
    'backup-storage-classes'
  );
  // Core is already deleting it; nothing left to change.
  if (row.status.state === BackupStorageState.Deleting) {
    return <TableActionsMenu menuItems={[]} />;
  }

  return (
    <TableActionsMenu
      menuItems={[
        ...(canUpdate && canReadClasses
          ? [
              <MenuItem key="edit" onClick={() => onEdit(row)}>
                <Edit /> {Messages.edit}
              </MenuItem>,
            ]
          : []),
        ...(canDelete
          ? [
              <MenuItem key="delete" onClick={() => onDelete(row)}>
                <Delete /> {Messages.delete}
              </MenuItem>,
            ]
          : []),
      ]}
    />
  );
};
