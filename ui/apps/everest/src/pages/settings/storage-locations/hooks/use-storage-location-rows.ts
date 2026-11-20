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

import { useBackupStorageClasses } from 'hooks/api/backup-storage-classes';
import { useBackupStorages } from 'hooks/api/backup-storages';
import { BackupStorageClass } from 'shared-types/backupStorages.types';
import { toBackupStorageRows } from '../storage-locations.utils';

const NO_CLASSES: BackupStorageClass[] = [];

// The storages of every namespace, each with its class's display name.
export const useStorageLocationRows = () => {
  const storageResults = useBackupStorages();
  const { data: classes = NO_CLASSES } = useBackupStorageClasses();

  return {
    rows: toBackupStorageRows(
      storageResults.flatMap((result) => result.data ?? []),
      classes
    ),
    isLoading: storageResults.some((result) => result.isLoading),
  };
};
