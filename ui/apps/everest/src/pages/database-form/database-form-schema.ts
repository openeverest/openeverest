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
import { MAX_DB_CLUSTER_NAME_LENGTH } from '../../consts.ts';
import { Messages } from './database-form.messages.ts';
import { DbWizardFormFields } from 'consts.ts';
import { rfc_123_schema } from 'utils/common-validation.ts';
import { DbInstanceName } from './database-form.types.ts';
import { Instance } from 'shared-types/api.types.ts';

const basicInfoFieldsSchema = z.object({
  [DbWizardFormFields.provider]: z.string(),
  [DbWizardFormFields.dbName]: rfc_123_schema({
    fieldName: 'database name',
  })
    .max(MAX_DB_CLUSTER_NAME_LENGTH, Messages.errors.dbName.tooLong)
    .nonempty(),
  [DbWizardFormFields.k8sNamespace]: z.string().nullable(),
  topology: z.object({ type: z.string() }),
  // UI-only picker; excluded from the Instance spec by NON_SPEC_FORM_FIELDS.
  [DbWizardFormFields.presetId]: z.string().optional(),
});

const basicInfoSchema = (dbInstances: DbInstanceName[]) =>
  basicInfoFieldsSchema
    .passthrough()
    .superRefine(({ dbName, k8sNamespace }, ctx) => {
      const dbInstancesInNamespaceNamesList = dbInstances.filter(
        (res) => res.namespace === k8sNamespace
      );

      if (
        dbInstancesInNamespaceNamesList.find((item) => item.name === dbName)
      ) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [DbWizardFormFields.dbName],
          message: Messages.errors.dbName.duplicate,
        });
      }
    });

// .passthrough tells Zod to not drop unrecognized keys
// this is needed because we parse step by step
// so, by default, Zod would leave behind the keys from previous steps

// const stepTwoSchema = (
//   defaultValues: Record<string, unknown>,
//   mode: WizardMode
// ) => resourcesFormSchema(defaultValues, mode === WizardMode.New, true, true);

// const backupsStepSchema = () =>
//   z
//     .object({
//       [DbWizardFormFields.schedules]: z.array(
//         z.object({
//           backupStorageName: z.string(),
//           enabled: z.boolean(),
//           name: z.string(),
//           schedule: z.string(),
//         })
//       ),
//       [DbWizardFormFields.pitrEnabled]: z.boolean(),
//       [DbWizardFormFields.pitrStorageLocation]: z
//         .string()
//         .or(
//           z.object({
//             name: z.string(),
//           })
//         )
//         .nullable()
//         .optional(),
//     })
//     .passthrough()
//     .superRefine(({ pitrEnabled, pitrStorageLocation }, ctx) => {
//       if (pitrEnabled && !pitrStorageLocation) {
//         ctx.addIssue({
//           code: z.ZodIssueCode.custom,
//           path: [DbWizardFormFields.pitrStorageLocation],
//           message: ScheduleFormMessages.storageLocation.invalidOption,
//         });
//       }
//     });

export const getDBWizardSchema = (
  dbInstances: DbInstanceName[],
  openApiValidationSchema?: z.ZodTypeAny,
  // Validates fields the static steps render from other schemas, e.g. the
  // monitoring entries' parameters.
  stepsValidationSchema?: z.ZodTypeAny
): z.ZodTypeAny => {
  const schemas = [
    basicInfoSchema(dbInstances),
    openApiValidationSchema,
    stepsValidationSchema,
  ];

  // These schemas retain other steps' values. Forward parsed output rather than
  // intersecting raw and transformed values, which cannot merge numeric coercion.
  return z.unknown().transform((data, ctx) => {
    let parsed: unknown = data;
    schemas.forEach((schema) => {
      if (!schema) return;
      const result = schema.safeParse(parsed);
      if (!result.success) {
        result.error.issues.forEach((issue) => ctx.addIssue(issue));
      } else {
        parsed = result.data;
      }
    });
    return parsed;
  });
};

export type BasicInfoType = z.infer<typeof basicInfoFieldsSchema>;
export type DbWizardFormSpec = Omit<NonNullable<Instance['spec']>, 'topology'>;

export type DbWizardTypeBase = BasicInfoType & {
  spec: DbWizardFormSpec;
};

export type DbWizardType = DbWizardTypeBase;
