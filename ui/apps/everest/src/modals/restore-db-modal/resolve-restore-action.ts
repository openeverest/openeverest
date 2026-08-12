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
  RestoreCreateVariables,
  RestoreDataSourceInput,
} from 'shared-types/restores.types';
import {
  BackupTypeValues,
  RecoveryTargetValues,
  RestoreDbFormData,
} from './restore-db-modal-schema';
import {
  RestorePitrStorageOption,
  RestorableBackupOption,
} from './restore-db-modal.types';
import { resolveActiveStorage, toRestoreDateISO } from './restore-pitr.utils';

// The decision a restore submission resolves to, independent of side effects:
// navigating to the create-new-DB flow (carrying the restore intent so the
// wizard can replay it against the freshly created instance), firing the
// restore mutation, or nothing (a state the schema already guards against).
export type RestoreAction =
  | { kind: 'navigate-new'; dataSource: RestoreDataSourceInput }
  | { kind: 'restore'; variables: RestoreCreateVariables }
  | { kind: 'none' };

interface RestoreActionContext {
  instanceName: string;
  isNewClusterMode: boolean;
  pitrStorages: RestorePitrStorageOption[];
  succeededBackups?: RestorableBackupOption[];
}

// Map validated form values to the restore data source, or undefined when the
// selection is incomplete (a state the schema already guards against). When
// cloning into a new DB the new instance has no stream of its own, so the
// source instance is named explicitly; an in-place restore leaves it implicit.
const resolveRestoreDataSourceInput = (
  data: RestoreDbFormData,
  {
    instanceName,
    isNewClusterMode,
    pitrStorages,
    succeededBackups,
  }: RestoreActionContext
): RestoreDataSourceInput | undefined => {
  if (data.backupType === BackupTypeValues.fromBackup) {
    if (!data.backupName) {
      return undefined;
    }
    // Carry the backup's storage so a clone can register it (the seeding
    // storage the operator reads from); ignored for an in-place restore.
    const storageName = succeededBackups?.find(
      (backup) => backup.name === data.backupName
    )?.storageName;
    return {
      type: 'Backup',
      backupName: data.backupName,
      ...(storageName ? { storageName } : {}),
    };
  }

  const storage = resolveActiveStorage(pitrStorages, data.pitrStorage);
  if (!storage) {
    return undefined;
  }

  const sourceInstanceName = isNewClusterMode ? instanceName : undefined;

  if (data.recoveryTarget === RecoveryTargetValues.date) {
    if (!data.pointInTimeDate) {
      return undefined;
    }
    return {
      type: 'PointInTime',
      storageName: storage.name,
      recoveryTarget: 'date',
      date: toRestoreDateISO(data.pointInTimeDate),
      ...(sourceInstanceName ? { sourceInstanceName } : {}),
    };
  }

  return {
    type: 'PointInTime',
    storageName: storage.name,
    recoveryTarget: 'latest',
    ...(sourceInstanceName ? { sourceInstanceName } : {}),
  };
};

// Pure mapping from validated form values to the restore action. Kept free of
// hooks and side effects so every branch (backup, PITR latest/date, in-place
// vs. clone-to-new-DB) is unit-testable in isolation.
export const resolveRestoreAction = (
  data: RestoreDbFormData,
  context: RestoreActionContext
): RestoreAction => {
  const dataSource = resolveRestoreDataSourceInput(data, context);
  if (!dataSource) {
    return { kind: 'none' };
  }

  if (context.isNewClusterMode) {
    return { kind: 'navigate-new', dataSource };
  }

  return {
    kind: 'restore',
    variables: { instanceName: context.instanceName, ...dataSource },
  };
};
