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
import { Messages } from './monitoring-destinations.messages';
import {
  MonitoringDestinationPermissions,
  MonitoringDestinationRow,
} from './monitoring-destinations.types';

export const getMonitoringDestinationMenuItems = ({
  row,
  permissions,
  onEdit,
  onDelete,
}: {
  row: MonitoringDestinationRow;
  permissions: MonitoringDestinationPermissions;
  onEdit: (row: MonitoringDestinationRow) => void;
  onDelete: (row: MonitoringDestinationRow) => void;
}) => [
  ...(permissions.updateNamespaces.includes(row.namespace)
    ? [
        <MenuItem key="edit" onClick={() => onEdit(row)}>
          <Edit /> {Messages.edit}
        </MenuItem>,
      ]
    : []),
  ...(permissions.deleteNamespaces.includes(row.namespace)
    ? [
        <MenuItem key="delete" onClick={() => onDelete(row)}>
          <Delete /> {Messages.delete}
        </MenuItem>,
      ]
    : []),
];
