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

import { z } from 'zod';
import { Messages } from './restore-db-modal.messages';
import { RestorePitrStorageOption } from './restore-db-modal.types';
import { resolveActiveStorage } from './restore-pitr.utils';

export enum RestoreDbFields {
  backupType = 'backupType',
  backupName = 'backupName',
  pitrStorage = 'pitrStorage',
  recoveryTarget = 'recoveryTarget',
  pointInTimeDate = 'pointInTimeDate',
}

export enum BackupTypeValues {
  fromBackup = 'fromBackup',
  fromPitr = 'fromPITR',
}

export enum RecoveryTargetValues {
  latest = 'latest',
  date = 'date',
}

export const schema = (pitrStorages: RestorePitrStorageOption[] = []) =>
  z
    .object({
      [RestoreDbFields.backupType]: z.nativeEnum(BackupTypeValues),
      [RestoreDbFields.backupName]: z.string().optional(),
      [RestoreDbFields.pitrStorage]: z.string().optional(),
      [RestoreDbFields.recoveryTarget]: z
        .nativeEnum(RecoveryTargetValues)
        .optional(),
      [RestoreDbFields.pointInTimeDate]: z.date().nullable().optional(),
    })
    .superRefine((data, ctx) => {
      if (data.backupType === BackupTypeValues.fromBackup) {
        if (!data.backupName) {
          ctx.addIssue({
            type: 'string',
            inclusive: true,
            code: z.ZodIssueCode.too_small,
            minimum: 1,
            path: [RestoreDbFields.backupName],
          });
        }
        return;
      }

      const storage = resolveActiveStorage(pitrStorages, data.pitrStorage);

      if (!storage) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [RestoreDbFields.pitrStorage],
          message: Messages.pitrNoStorage,
        });
        return;
      }

      if (!storage.window.available) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [RestoreDbFields.pitrStorage],
          message: storage.window.message ?? Messages.pitrUnavailable,
        });
        return;
      }

      if (data.recoveryTarget !== RecoveryTargetValues.date) {
        return;
      }

      if (!data.pointInTimeDate) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [RestoreDbFields.pointInTimeDate],
          message: Messages.pitrDateRequired,
        });
        return;
      }

      const { earliest, latest } = storage.window;
      const outOfRange =
        (earliest && data.pointInTimeDate < earliest) ||
        (latest && data.pointInTimeDate > latest);
      if (outOfRange) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [RestoreDbFields.pointInTimeDate],
          message: Messages.pitrDateOutOfRange,
        });
      }
    });

export const defaultValues = {
  [RestoreDbFields.backupType]: BackupTypeValues.fromBackup,
  [RestoreDbFields.backupName]: '',
  [RestoreDbFields.pitrStorage]: '',
  [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.latest,
  [RestoreDbFields.pointInTimeDate]: null,
};

export type RestoreDbFormData = z.infer<ReturnType<typeof schema>>;
