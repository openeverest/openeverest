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

import { Tooltip, Typography } from '@mui/material';
import StatusField from 'components/status-field';
import {
  BackupStorageState,
  BackupStorageStatus,
} from 'shared-types/backupStorages.types';
import { describeBackupStorageStatus } from 'utils/backup-storages';
import { BackupStorageStateMessages } from 'utils/backup-storages.messages';
import { BACKUP_STORAGE_STATE_TO_STATUS } from '../storage-locations.constants';

interface StorageStatusCellProps {
  status: BackupStorageStatus;
}

// The state, with what is wrong beneath it and core's own message on hover.
export const StorageStatusCell = ({ status }: StorageStatusCellProps) => {
  const description = describeBackupStorageStatus(status);
  const explained =
    status.state === BackupStorageState.Invalid ||
    status.state === BackupStorageState.Deleting;
  const tooltip = explained
    ? status.message !== description && status.message
    : description;

  return (
    <Tooltip title={tooltip || ''} placement="top">
      <span>
        <StatusField
          status={status.state}
          statusMap={BACKUP_STORAGE_STATE_TO_STATUS}
        >
          <Typography variant="body2">
            {BackupStorageStateMessages[status.state]}
          </Typography>
        </StatusField>
        {explained && description && (
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ display: 'block' }}
          >
            {description}
          </Typography>
        )}
      </span>
    </Tooltip>
  );
};
