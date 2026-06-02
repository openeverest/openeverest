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

import { AutoCompleteAutoFillProps } from 'components/auto-complete-auto-fill/auto-complete-auto-fill.types';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Instance } from 'shared-types/api.types';

// Minimal schedule shape required by BackupStoragesInput for limit calculations
export interface ScheduleWithStorage {
  backupStorageName: string;
}

export type InstanceStorage = NonNullable<
  NonNullable<Instance['spec']['backup']>['storages']
>[number];

export type BackupStoragesInputProps = {
  name?: string;
  namespace: string;
  // Controls how the backup storage field behaves in different product flows.
  /* - On-demand and schedule creation forms: keep default auto-selection of the first
   *   available storage to reduce user actions.
   * - Schedule edit and similar update flows: disable auto-selection to preserve
   *   the storage already saved in the existing configuration.
   */
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorageCRD>>;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
} & (
  | {
      /**
       * Nested instance storages (v2 model). When provided, `schedules` and
       * `instanceStorageNames` are derived internally and should NOT be passed.
       */
      instanceStorages: InstanceStorage[];
      schedules?: never;
      instanceStorageNames?: never;
    }
  | {
      /** Flat schedules list (wizard/legacy path). */
      schedules: ScheduleWithStorage[];
      /** Storage names currently active on the instance. */
      instanceStorageNames?: string[];
      instanceStorages?: never;
    }
);
