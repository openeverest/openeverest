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

import { isSection } from 'components/ui-generator/ui-generator.types';
import {
  BackupStorageClass,
  BackupStorageClassUiSchema,
  BackupStorageCondition,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';

export const BACKUP_STORAGE_ACCEPTED_CONDITION = 'Accepted';
export const BACKUP_STORAGE_DELETING_CONDITION = 'Deleting';

export const getBackupStorageCondition = (
  storage: BackupStorageCRD,
  type: string
): BackupStorageCondition | undefined =>
  storage.status?.conditions?.find((condition) => condition.type === type);

export const formatBackupStorageLocation = ({
  spec: { location },
}: BackupStorageCRD): string =>
  location.prefix ? `${location.bucket}/${location.prefix}` : location.bucket;

export const getBackupStorageClassDisplayName = (
  backupStorageClass: BackupStorageClass | undefined,
  name: string
): string => backupStorageClass?.spec.displayName || name;

export const getBackupStorageClassUiSchema = (
  backupStorageClass: BackupStorageClass | undefined
): BackupStorageClassUiSchema => {
  const uiSchema: Record<string, unknown> =
    backupStorageClass?.spec.uiSchema ?? {};
  const credentials: Record<string, unknown> =
    typeof uiSchema.credentials === 'object' && uiSchema.credentials !== null
      ? (uiSchema.credentials as Record<string, unknown>)
      : {};

  return {
    ...(isSection(uiSchema.location) && { location: uiSchema.location }),
    ...(isSection(uiSchema.storage) && { storage: uiSchema.storage }),
    credentials: Object.fromEntries(
      Object.entries(credentials).filter(([, section]) => isSection(section))
    ) as BackupStorageClassUiSchema['credentials'],
  };
};
