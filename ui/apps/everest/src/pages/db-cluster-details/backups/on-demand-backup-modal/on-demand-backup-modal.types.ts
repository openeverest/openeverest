import z from 'zod';
import { generateShortUID } from 'utils/generateShortUID';
import { rfc_123_schema } from 'utils/common-validation';
import type { Section } from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { FormMode } from 'components/ui-generator/ui-generator.types';

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
    [BackupFields.name]: rfc_123_schema({ fieldName: 'backup name' })
      .superRefine((input, ctx) => {
        if (backupsNamesList.find((item) => item === input)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: 'You already have a backup with this name',
          });
        }
      }),
    [BackupFields.backupClassName]: z
      .string()
      .min(1, 'Backup class is required'),
    [BackupFields.storageName]: z
      .string()
      .or(z.object({ name: z.string() }))
      .nullish()
      .transform((v) => {
        if (v == null) return '';
        return typeof v === 'string' ? v : v.name;
      })
      .pipe(z.string().min(1, 'Storage is required')),
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
