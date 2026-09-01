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
import { flattenSchedules } from 'utils/backup-schedules';
import { Instance } from 'shared-types/api.types';
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
          ...(schedule.parameters ? { parameters: schedule.parameters } : {}),
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

/**
 * Inverse of buildBackupSpecFromWizard: project an existing Instance's nested
 * spec.backup onto the wizard's flat form state so a clone (restore-to-new-DB)
 * inherits the source's schedules, backup class, and per-storage PITR — matching
 * the create wizard's restore prefill.
 */
export const extractWizardBackup = (
  instance: Instance
): {
  schedules: FlattenedSchedule[];
  classRef: { name: string };
  pitr: WizardPitrMap;
} => {
  const storages = instance.spec?.backup?.storages ?? [];
  const pitr: WizardPitrMap = {};
  for (const storage of storages) {
    if (storage.pitr?.enabled) {
      pitr[storage.storageRef.name ?? ''] = {
        enabled: true,
        ...(storage.pitr.parameters
          ? { parameters: storage.pitr.parameters as Record<string, unknown> }
          : {}),
      };
    }
  }

  return {
    schedules: flattenSchedules(instance),
    classRef: { name: instance.spec?.backup?.classRef?.name ?? '' },
    pitr,
  };
};

/**
 * Guarantee the seeding storage (where a clone reads its source data) is present
 * in the instance's backup spec with backup enabled, independent of the user's
 * schedule choices. Without it the operator stalls waiting for the storage.
 * Returns the spec unchanged when there is no storage to register.
 */
export const ensureStorageRegistered = (
  spec: WizardBackupSpec | undefined,
  storageName: string | undefined,
  classRefName: string | undefined
): WizardBackupSpec | undefined => {
  if (!storageName) {
    return spec;
  }

  if (spec) {
    if (
      spec.storages.some((storage) => storage.storageRef.name === storageName)
    ) {
      return spec;
    }
    return {
      ...spec,
      storages: [
        ...spec.storages,
        { name: storageName, storageRef: { name: storageName }, schedules: [] },
      ],
    };
  }

  // No wizard backup spec (e.g. the user cleared every schedule): build a
  // minimal one that just registers the seeding storage. Without a class we
  // cannot form a valid spec, so leave it to the BE to reject.
  if (!classRefName) {
    return spec;
  }
  return {
    classRef: { name: classRefName },
    enabled: true,
    storages: [
      { name: storageName, storageRef: { name: storageName }, schedules: [] },
    ],
  };
};
