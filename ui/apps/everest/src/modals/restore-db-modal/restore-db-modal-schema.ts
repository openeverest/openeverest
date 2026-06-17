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

export enum RestoreDbFields {
  backupType = 'backupType',
  backupName = 'backupName',
  // pitrBackup = 'pitrBackup',
}

export enum BackupTypeValues {
  fromBackup = 'fromBackup',
  fromPitr = 'fromPITR',
}

// TODO: Re-enable PITR params (gaps: boolean, minDate?: Date, maxDate?: Date)
// when PITR restore flow is implemented.
export const schema = () =>
  z
    .object({
      [RestoreDbFields.backupType]: z.nativeEnum(BackupTypeValues),
      [RestoreDbFields.backupName]: z.string().optional(),
      // [RestoreDbFields.pitrBackup]: z.string().or(z.date()).optional(),
    })
    .superRefine(({ backupType, backupName /*, pitrBackup */ }, ctx) => {
      if (backupType === BackupTypeValues.fromBackup) {
        if (!backupName) {
          ctx.addIssue({
            type: 'string',
            inclusive: true,
            code: z.ZodIssueCode.too_small,
            minimum: 1,
          });
        }
      }
      /* else {
        if (isDate(minDate) && isDate(maxDate) && !!pitrBackup) {
          const pitrBackupDate = isDate(pitrBackup)
            ? pitrBackup
            : new Date(pitrBackup);
          if (
            isAfter(pitrBackupDate, maxDate) ||
            isBefore(pitrBackupDate, minDate)
          ) {
            ctx.addIssue({
              code: z.ZodIssueCode.invalid_date,
            });
          }
        } else {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
          });
        }

        if (gaps) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
          });
        }

        if (!pitrBackup) {
          ctx.addIssue({
            code: z.ZodIssueCode.invalid_date,
          });
        }
      }
      */
    });

export const defaultValues = {
  [RestoreDbFields.backupType]: BackupTypeValues.fromBackup,
  [RestoreDbFields.backupName]: '',
  // [RestoreDbFields.pitrBackup]: '',
};

export type RestoreDbFormData = z.infer<ReturnType<typeof schema>>;
