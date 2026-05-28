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
import { Instance } from 'shared-types/api.types';
import { Backup } from 'shared-types/backups.types';

export const flattenSchedules = (instance: Instance): FlattenedSchedule[] =>
  (instance.spec?.backup?.storages ?? []).flatMap((storage) =>
    (storage.schedules ?? []).map((schedule) => ({
      name: schedule.name,
      cron: schedule.cron,
      enabled: schedule.enabled,
      retentionCopies: schedule.retentionCopies,
      config: schedule.config as Record<string, unknown> | undefined,
      storageName: storage.storageRef.name ?? '',
    }))
  );

export const applySchedulesToStorages = (
  instance: Instance,
  schedules: FlattenedSchedule[]
): NonNullable<
  NonNullable<NonNullable<Instance['spec']>['backup']>['storages']
> => {
  const existingStorages = instance.spec?.backup?.storages ?? [];
  return existingStorages.map((storage) => ({
    ...storage,
    schedules: schedules
      .filter((s) => s.storageName === (storage.storageRef.name ?? ''))
      .map((schedule) => ({
        name: schedule.name,
        cron: schedule.cron,
        enabled: schedule.enabled,
        retentionCopies: schedule.retentionCopies,
        ...(schedule.config
          ? { config: schedule.config as Record<string, never> }
          : {}),
      })),
  }));
};

/**
 * Removes storage entries from the instance that have no schedules
 * and no active Backup CRs referencing them.
 */
export const removeUnusedStorages = (
  storages: NonNullable<
    NonNullable<NonNullable<Instance['spec']>['backup']>['storages']
  >,
  activeBackups: Backup[]
) =>
  storages.filter((storage) => {
    const hasSchedules = (storage.schedules ?? []).length > 0;
    const hasBackups = activeBackups.some(
      (b) => b.spec?.storageName === storage.name
    );
    return hasSchedules || hasBackups;
  });
