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

import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { WizardBackupSpec, WizardPitrMap } from './backup-step.types';

/**
 * Transform the wizard's flat backup form state into the nested
 * Instance spec.backup shape expected by the API.
 *
 * Form state:
 *   backup.schedules: FlattenedSchedule[]
 *   backup.classRef.name: string
 *   backup.pitr: WizardPitrMap (per-storage, keyed by storage name)
 *
 * API shape:
 *   spec.backup: { classRef: { name }, enabled, storages: [{ name, storageRef, schedules, pitr? }] }
 */
export const buildBackupSpecFromWizard = (
  flatSchedules: FlattenedSchedule[],
  classRefName: string | undefined,
  pitrMap: WizardPitrMap = {}
): WizardBackupSpec | undefined => {
  if (!flatSchedules.length || !classRefName) return undefined;

  const storageMap = new Map<string, FlattenedSchedule[]>();
  for (const schedule of flatSchedules) {
    const existing = storageMap.get(schedule.storageName) ?? [];
    existing.push(schedule);
    storageMap.set(schedule.storageName, existing);
  }

  const storages = Array.from(storageMap.entries()).map(
    ([storageName, schedules]) => {
      // PITR is attached only for storages the user enabled it on; entries for
      // storages without schedules never reach this map (derived from schedules).
      const pitr = pitrMap[storageName];
      return {
        name: storageName,
        storageRef: { name: storageName },
        schedules: schedules.map((schedule) => ({
          name: schedule.name,
          cron: schedule.cron,
          enabled: schedule.enabled,
          ...(schedule.retentionCopies != null
            ? { retentionCopies: schedule.retentionCopies }
            : {}),
          ...(schedule.parameters
            ? { parameters: schedule.parameters }
            : {}),
        })),
        ...(pitr?.enabled
          ? {
              pitr: {
                enabled: true,
                ...(pitr.parameters ? { parameters: pitr.parameters } : {}),
              },
            }
          : {}),
      };
    }
  );

  return {
    classRef: { name: classRefName },
    enabled: true,
    storages,
  };
};
