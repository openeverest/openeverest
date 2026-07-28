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

import { InstanceBackupStorage } from './backups.types';

export const setStoragePitr = (
  storages: InstanceBackupStorage[],
  storageName: string,
  pitr: InstanceBackupStorage['pitr']
): InstanceBackupStorage[] =>
  storages.map((storage) =>
    storage.storageRef.name === storageName ? { ...storage, pitr } : storage
);

export const countPitrEnabledStorages = (
  storages: InstanceBackupStorage[]
): number => storages.filter((storage) => storage.pitr?.enabled).length;

export const hasActiveSchedules = (
  storages: InstanceBackupStorage[]
): boolean =>
  storages.some((storage) =>
    (storage.schedules ?? []).some((schedule) => schedule.enabled)
  );
