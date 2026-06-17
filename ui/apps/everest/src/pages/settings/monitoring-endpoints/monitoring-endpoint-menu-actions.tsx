// everest
// Copyright (C) 2023 Percona LLC
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

import { MRT_Row } from 'material-react-table';
import { MenuItem } from '@mui/material';
import { Delete, Edit } from '@mui/icons-material';
import { Messages } from './monitoring-endpoints.messages';
import { MonitoringConfigTableElement } from './monitoring-endpoints.types';
import { useRBACPermissions } from 'hooks/rbac';

export const MonitoringActionButtons = (
  row: MRT_Row<MonitoringConfigTableElement>,
  handleDeleteConfig: (instance: MonitoringConfigTableElement) => void,
  handleOpenEditModal: (instance: MonitoringConfigTableElement) => void
) => {
  const { canUpdate, canDelete } = useRBACPermissions(
    'monitoring-configs',
    `${row.original.namespace}/${row.original.name}`
  );
  return [
    ...(canUpdate
      ? [
          <MenuItem
            key={0}
            onClick={() => {
              handleOpenEditModal(row.original);
            }}
            sx={{
              m: 0,
              gap: 1,
              px: 2,
              py: '10px',
            }}
          >
            <Edit /> {Messages.edit}
          </MenuItem>,
        ]
      : []),
    ...(canDelete
      ? [
          <MenuItem
            key={1}
            onClick={() => {
              handleDeleteConfig(row.original);
            }}
          >
            <Delete /> {Messages.delete}
          </MenuItem>,
        ]
      : []),
  ];
};
