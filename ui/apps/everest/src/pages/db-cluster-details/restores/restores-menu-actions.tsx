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
import { Restore } from 'shared-types/restores.types';
import DeleteIcon from '@mui/icons-material/Delete';
import { useRBACPermissions } from 'hooks/rbac';

export const RestoreActionButtons = (
  row: MRT_Row<Restore>,
  handleDeleteRestore: (restoreName: string) => void,
  namespace: string,
  dbClusterName: string
) => {
  const { canDelete } = useRBACPermissions(
    'database-cluster-restores',
    `${namespace}/${dbClusterName}`
  );

  return [
    ...(canDelete
      ? [
          <MenuItem
            key={2}
            onClick={() => {
              handleDeleteRestore(row.original.name);
            }}
            sx={{ m: 0 }}
          >
            <DeleteIcon />
            Delete
          </MenuItem>,
        ]
      : []),
  ];
};
