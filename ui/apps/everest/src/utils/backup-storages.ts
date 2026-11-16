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

import {
  BackupStorageClass,
  BackupStorageCondition,
  BackupStorageCRD,
  BackupStorageState,
  BackupStorageStatus,
} from 'shared-types/backupStorages.types';
import { Messages } from './backup-storages.messages';

const ACCEPTED_CONDITION = 'Accepted';
const DELETING_CONDITION = 'Deleting';

const findCondition = (storage: BackupStorageCRD, type: string) =>
  storage.status?.conditions?.find((condition) => condition.type === type);

const explainedBy = (
  state: BackupStorageState,
  condition: BackupStorageCondition | undefined
): BackupStorageStatus => ({
  state,
  reason: condition?.reason || undefined,
  message: condition?.message || undefined,
});

export const getBackupStorageStatus = (
  storage: BackupStorageCRD
): BackupStorageStatus => {
  const deleting = findCondition(storage, DELETING_CONDITION);
  // Core holds the deletion until nothing uses the storage.
  if (storage.metadata?.deletionTimestamp || deleting?.status === 'True') {
    return explainedBy(BackupStorageState.Deleting, deleting);
  }
  const accepted = findCondition(storage, ACCEPTED_CONDITION);
  if (accepted?.status === 'True') {
    return { state: BackupStorageState.Valid };
  }
  return explainedBy(
    accepted?.status === 'False'
      ? BackupStorageState.Invalid
      : BackupStorageState.Pending,
    accepted
  );
};

const isKnownReason = (
  reason: string
): reason is keyof typeof Messages.reasons => reason in Messages.reasons;

// What is wrong, in the user's terms when core's reason is a known one.
export const describeBackupStorageStatus = ({
  state,
  reason,
  message,
}: BackupStorageStatus): string | undefined => {
  if (state === BackupStorageState.Valid) {
    return Messages.notTested;
  }
  if (reason && isKnownReason(reason)) {
    return Messages.reasons[reason];
  }
  return (
    message ||
    reason ||
    (state === BackupStorageState.Pending ? Messages.notCheckedYet : undefined)
  );
};

export const formatBackupStorageLocation = ({
  spec: {
    location: { objectStore },
  },
}: BackupStorageCRD) =>
  objectStore.prefix
    ? `${objectStore.bucket}/${objectStore.prefix}`
    : objectStore.bucket;

export const getBackupStorageClassTitle = (
  className: string,
  classes: BackupStorageClass[]
) =>
  classes.find((storageClass) => storageClass.metadata?.name === className)
    ?.spec.displayName || className;
