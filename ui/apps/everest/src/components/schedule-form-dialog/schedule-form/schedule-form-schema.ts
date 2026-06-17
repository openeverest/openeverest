// everest
// Copyright (C) 2023 Percona LLC
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

import z from 'zod';
import { MAX_SCHEDULE_NAME_LENGTH } from '../../../consts.ts';
import { Messages } from './schedule-form.messages.ts';
import { ScheduleFormFields } from './schedule-form.types.ts';
import { rfc_123_schema } from 'utils/common-validation';
import { timeSelectionSchemaObject } from '../../time-selection/time-selection-schema.ts';
import { FlattenedSchedule } from '../schedule-form-dialog-context/schedule-form-dialog-context.types';
import { getCronExpressionFromFormValues } from '../../time-selection/time-selection.utils';
import { sameScheduleFunc } from '../schedule-form-dialog.utils';
import { WizardMode } from 'shared-types/wizard.types.ts';

export const storageLocationZodObject = z
  .string()
  .or(
    z
      .object({
        metadata: z.object({ name: z.string() }).passthrough(),
      })
      .passthrough()
  )
  .nullable();

export const storageLocationScheduleFormSchema = (
  mode: 'dbWizard' | 'scheduledBackups' | 'pitr'
) => {
  return {
    [ScheduleFormFields.storageLocation]: storageLocationZodObject.superRefine(
      (input, ctx) => {
        // TODO revert next line check after https://jira.percona.com/browse/EVEREST-509
        //  this is a temporary measure, as soon as PostgresSQL is implemented, the StorageLocation check
        //  will become mandatory everywhere and it will be possible to remove the null check at all,
        const checkNullStorage = mode !== 'dbWizard';
        if (
          (!input || typeof input === 'string' || !input.metadata?.name) &&
          (checkNullStorage ? true : input !== null)
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: Messages.storageLocation.invalidOption,
          });
        }
      }
    ),
  };
};

export const schema = (schedules: FlattenedSchedule[], mode: WizardMode) => {
  const schedulesNamesList = schedules.map((item) => item?.name);
  return z
    .object({
      [ScheduleFormFields.scheduleName]: rfc_123_schema({
        fieldName: `${Messages.scheduleName.label.toLowerCase()} name`,
      })
        .nonempty()
        .max(MAX_SCHEDULE_NAME_LENGTH, Messages.scheduleName.tooLong)
        .superRefine((input, ctx) => {
          if (
            mode === WizardMode.New &&
            !!schedulesNamesList.find((item) => item === input)
          ) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: Messages.scheduleName.duplicate,
            });
          }
        }),
      [ScheduleFormFields.retentionCopies]: z
        .string()
        .superRefine((nrCopies, ctx) => {
          const nrCopiesInt = parseInt(nrCopies, 10);

          if (
            isNaN(nrCopiesInt) ||
            nrCopiesInt < 0 ||
            nrCopiesInt > Math.pow(2, 31) - 1
          ) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: Messages.retentionCopies.invalidNumber,
            });
          }
        }),
      [ScheduleFormFields.backupClassName]: z
        .string()
        .min(1, Messages.backupClass.required),
      ...timeSelectionSchemaObject,
      ...storageLocationScheduleFormSchema('scheduledBackups'),
    })
    .passthrough()
    .superRefine(
      (
        { selectedTime, hour, minute, onDay, weekDay, amPm, scheduleName },
        ctx
      ) => {
        const currentSchedule = getCronExpressionFromFormValues({
          selectedTime,
          amPm,
          hour,
          minute,
          onDay,
          weekDay,
        });
        const sameSchedule = sameScheduleFunc(
          schedules,
          mode,
          currentSchedule,
          scheduleName
        );
        if (sameSchedule) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: Messages.sameTimeSchedule,
            path: ['root'],
          });
        }
      }
    );
};

export type ScheduleFormData = z.infer<ReturnType<typeof schema>>;
