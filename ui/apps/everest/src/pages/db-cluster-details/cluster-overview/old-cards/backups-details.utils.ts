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

import { format } from 'date-fns';
import { DATE_FORMAT } from 'consts';
import { Instance } from 'shared-types/api.types';
import { Backup } from 'shared-types/backups.types';

/** Formats a backup timestamp, falling back to an em dash when it is absent. */
export const formatBackupDate = (value?: string): string =>
  value ? format(value, DATE_FORMAT) : '\u2014';

/** Names of the instance's backup storages that have PITR enabled. */
export const getPitrEnabledStorageNames = (instance: Instance): string[] =>
  (instance.spec?.backup?.storages ?? [])
    .filter((storage) => storage.pitr?.enabled)
    .map((storage) => storage.storageRef.name ?? '')
    .filter(Boolean);

/** Most recent backups first (by started, then completed time), capped at `limit`. */
export const getRecentBackups = (backups: Backup[], limit: number): Backup[] =>
  [...backups]
    .sort((a, b) => {
      const aDate = a.status?.startedAt ?? a.status?.completedAt ?? '';
      const bDate = b.status?.startedAt ?? b.status?.completedAt ?? '';
      return bDate.localeCompare(aDate);
    })
    .slice(0, limit);
