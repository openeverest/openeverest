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

import { Alert, Stack } from '@mui/material';
import { useContext } from 'react';
import { ScheduleModalContext } from '../backups.context';
import { StoragePitrToggle } from './storage-pitr-toggle';
import { useShowPitrBackupWarning } from './use-show-pitr-backup-warning';
import { Messages } from './storages-list.messages';

export const StoragesList = () => {
  const { instance } = useContext(ScheduleModalContext);
  const storages = instance.spec?.backup?.storages ?? [];
  const showNeedsBackupWarning = useShowPitrBackupWarning(instance);

  if (storages.length === 0) {
    return null;
  }

  return (
    <Stack
      useFlexGap
      spacing={1}
      data-testid="storages-list"
      sx={{
        width: '100%',
        order: 3,
        bgcolor: (theme) => theme.palette.surfaces?.elevation0,
        p: 2,
        mt: 1,
      }}
    >
      {showNeedsBackupWarning && (
        <Alert severity="info" data-testid="pitr-needs-backup-warning">
          {Messages.needsBackup}
        </Alert>
      )}
      {storages.map((storage) => (
        <StoragePitrToggle key={storage.storageRef.name} storage={storage} />
      ))}
    </Stack>
  );
};
