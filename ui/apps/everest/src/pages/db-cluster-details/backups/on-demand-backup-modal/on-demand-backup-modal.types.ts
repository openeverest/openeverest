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
import { generateShortUID } from 'utils/generateShortUID';
import { rfc_123_schema } from 'utils/common-validation';
import type { Section } from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { Messages } from './on-demand-backup-modal.messages';

export enum BackupFields {
  name = 'name',
  backupClassName = 'backupClassName',
  storageName = 'storageName',
}

export const defaultValuesFc = () => ({
  [BackupFields.name]: `backup-${generateShortUID()}`,
  [BackupFields.backupClassName]: '',
  [BackupFields.storageName]: undefined,
});

const staticSchema = (backupsNamesList: string[]) =>
  z.object({
    [BackupFields.name]: rfc_123_schema({
      fieldName: 'backup name',
    }).superRefine((input, ctx) => {
      if (backupsNamesList.find((item) => item === input)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: Messages.duplicateBackupName,
        });
      }
    }),
    [BackupFields.backupClassName]: z
      .string()
      .min(1, Messages.backupClassRequired),
    [BackupFields.storageName]: z
      .string()
      .or(z.object({ name: z.string() }))
      .nullish()
      // AutoComplete can return either a string or { name: string } object.
      // In v1 this normalization was done in the hook (backupStorageName: typeof ... === 'string' ? ... : ....name).
      // Now we handle it at the schema level via transform.
      .transform((v) => {
        if (v == null) return '';
        return typeof v === 'string' ? v : v.name;
      })
      .pipe(z.string().min(1, Messages.storageRequired)),
  });

/**
 * Builds a combined zod schema: static fields + dynamic UIGenerator fields.
 * When `configSections` is provided, UIGenerator field validation is merged in.
 * Otherwise, falls back to the static schema with `.passthrough()` to allow
 * dynamic fields to pass without validation.
 */
export const schema = (
  backupsNamesList: string[],
  configSections?: Record<string, Section>
) => {
  const base = staticSchema(backupsNamesList);

  if (!configSections) {
    return base.passthrough();
  }

  const { schema: dynamicSchema } = buildSectionZodSchema(
    'config',
    configSections,
    { formMode: FormMode.New }
  );

  // ZodIntersection (.and()) fails when `base` contains .transform() fields (storageName).
  // Validate UIGenerator fields separately via superRefine to avoid merge conflicts.
  return base.passthrough().superRefine((data, ctx) => {
    const result = dynamicSchema.safeParse(data);
    if (!result.success) {
      for (const issue of result.error.issues) {
        ctx.addIssue(issue);
      }
    }
  });
};

export type BackupFormData = z.infer<ReturnType<typeof staticSchema>> &
  Record<string, unknown>;
