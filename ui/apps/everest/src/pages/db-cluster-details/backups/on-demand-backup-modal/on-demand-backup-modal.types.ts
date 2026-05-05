import z from 'zod';
import { generateShortUID } from 'utils/generateShortUID';

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

export const schema = (backupsNamesList: string[]) =>
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

export type BackupFormData = z.infer<ReturnType<typeof schema>>;
