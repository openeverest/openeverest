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
  RestoreDataSource,
  RestoreDataSourceInput,
} from 'shared-types/restores.types';

const isStringOrUndefined = (value: unknown): value is string | undefined =>
  value === undefined || typeof value === 'string';

// Build the Restore CRD dataSource from an FE input. storageRef is always sent
// (the BE never infers it); date is included only for a date target and omitted
// for 'latest' (the schema forbids it there). instanceRef is set only when
// cloning (sourceInstanceName present); an in-place restore omits it so the BE
// defaults to the target instance's own stream.
export const buildRestoreDataSource = (
  input: RestoreDataSourceInput
): RestoreDataSource => {
  if (input.type === 'Backup') {
    return {
      type: 'Backup',
      backup: { backupRef: { name: input.backupName } },
    };
  }

  return {
    type: 'PointInTime',
    pointInTime: {
      recoveryTarget: input.recoveryTarget,
      source: {
        storageRef: { name: input.storageName },
        ...(input.sourceInstanceName
          ? { instanceRef: { name: input.sourceInstanceName } }
          : {}),
      },
      ...(input.recoveryTarget === 'date' ? { date: input.date } : {}),
    },
  };
};

// Runtime shape check for the restore intent carried through untyped router
// state. Validates the discriminant and its required fields before the value is
// trusted as a restore payload.
export const isRestoreDataSourceInput = (
  value: unknown
): value is RestoreDataSourceInput => {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const type = Reflect.get(value, 'type');

  if (type === 'Backup') {
    return (
      typeof Reflect.get(value, 'backupName') === 'string' &&
      isStringOrUndefined(Reflect.get(value, 'storageName'))
    );
  }

  if (type === 'PointInTime') {
    const storageName = Reflect.get(value, 'storageName');
    if (typeof storageName !== 'string') {
      return false;
    }

    if (!isStringOrUndefined(Reflect.get(value, 'sourceInstanceName'))) {
      return false;
    }

    const recoveryTarget = Reflect.get(value, 'recoveryTarget');
    if (recoveryTarget === 'latest') {
      return true;
    }

    return (
      recoveryTarget === 'date' &&
      typeof Reflect.get(value, 'date') === 'string'
    );
  }

  return false;
};
