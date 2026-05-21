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
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';

export type BackupStoragesInputProps = {
  name?: string;
  namespace: string;
  schedules: Schedule[];
  /**
   * Controls how the backup storage field behaves in different product flows.
   *
   * Typical usage:
   * - On-demand and schedule creation forms: keep default auto-selection of the first
   *   available storage to reduce user actions.
   * - Schedule edit and similar update flows: disable auto-selection to preserve
   *   the storage already saved in the existing configuration.
   */
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorage>>;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
  /** Storage names currently active on the instance (instance.spec.backup.storages[].storageRef.name). */
  instanceStorageNames?: string[];
  // TODO: schedules — hideUsedStoragesInSchedules was used for PostgreSQL
  // to hide storages already assigned to other schedules (PG slot limit).
  // Re-enable when schedule feature is implemented.
  // hideUsedStoragesInSchedules?: boolean;
};
