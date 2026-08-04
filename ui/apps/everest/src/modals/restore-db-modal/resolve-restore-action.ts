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

import { RestoreCreateVariables } from 'shared-types/restores.types';
import {
  BackupTypeValues,
  RecoveryTargetValues,
  RestoreDbFormData,
} from './restore-db-modal-schema';
import { RestorePitrStorageOption } from './restore-db-modal.types';
import { resolveActiveStorage, toRestoreDateISO } from './restore-pitr.utils';

// The decision a restore submission resolves to, independent of side effects:
// navigating to the create-new-DB flow, firing the restore mutation, or nothing
// (a state the schema already guards against).
export type RestoreAction =
  | { kind: 'navigate-new'; backupName: string }
  | { kind: 'restore'; variables: RestoreCreateVariables }
  | { kind: 'none' };

interface RestoreActionContext {
  instanceName: string;
  isNewClusterMode: boolean;
  pitrStorages: RestorePitrStorageOption[];
}

// Pure mapping from validated form values to the restore action. Kept free of
// hooks and side effects so every branch (backup, seed-new, PITR latest/date)
// is unit-testable in isolation.
export const resolveRestoreAction = (
  data: RestoreDbFormData,
  { instanceName, isNewClusterMode, pitrStorages }: RestoreActionContext
): RestoreAction => {
  if (data.backupType === BackupTypeValues.fromBackup) {
    if (!data.backupName) {
      return { kind: 'none' };
    }
    if (isNewClusterMode) {
      return { kind: 'navigate-new', backupName: data.backupName };
    }
    return {
      kind: 'restore',
      variables: { instanceName, type: 'Backup', backupName: data.backupName },
    };
  }

  const storage = resolveActiveStorage(pitrStorages, data.pitrStorage);
  if (!storage) {
    return { kind: 'none' };
  }

  if (data.recoveryTarget === RecoveryTargetValues.date) {
    if (!data.pointInTimeDate) {
      return { kind: 'none' };
    }
    return {
      kind: 'restore',
      variables: {
        instanceName,
        type: 'PointInTime',
        storageName: storage.name,
        recoveryTarget: 'date',
        date: toRestoreDateISO(data.pointInTimeDate),
      },
    };
  }

  return {
    kind: 'restore',
    variables: {
      instanceName,
      type: 'PointInTime',
      storageName: storage.name,
      recoveryTarget: 'latest',
    },
  };
};
