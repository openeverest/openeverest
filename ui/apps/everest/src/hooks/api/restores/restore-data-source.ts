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

// Build the Restore CRD dataSource from an FE input. storageRef is always sent
// (the BE never infers it); date is included only for a date target and omitted
// for 'latest' (the schema forbids it there).
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
      source: { storageRef: { name: input.storageName } },
      ...(input.recoveryTarget === 'date' ? { date: input.date } : {}),
    },
  };
};
