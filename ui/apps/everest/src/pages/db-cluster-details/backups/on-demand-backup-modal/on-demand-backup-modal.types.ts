import z from 'zod';
import { generateShortUID } from 'utils/generateShortUID';
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
  [BackupFields.storageName]: '',
});

const staticSchema = (backupsNamesList: string[]) =>
  z.object({
    [BackupFields.name]: z
      .string()
      .min(1)
      .superRefine((input, ctx) => {
        if (backupsNamesList.find((item) => item === input)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: 'You already have a backup with this name',
          });
        }
      }),
    [BackupFields.backupClassName]: z.string().min(1, 'Backup class is required'),
    [BackupFields.storageName]: z.string().min(1, 'Storage is required'),
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

  return base.and(dynamicSchema);
};

export type BackupFormData = z.infer<ReturnType<typeof staticSchema>> &
  Record<string, unknown>;
