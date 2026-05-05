import { z } from 'zod';
import { ImportFields } from './import.types';

const MAX_IMPORT_FIELD_LENGTH = 250;

const basicStringValidation = () =>
  z.string().trim().min(1).max(MAX_IMPORT_FIELD_LENGTH);

const dataImporterSchemeObject = {
  [ImportFields.dataImporter]: z.string().max(MAX_IMPORT_FIELD_LENGTH),
};

export const dataImporterSchema = z.object(dataImporterSchemeObject);

const s3SchemaObject = {
  [ImportFields.bucketName]: basicStringValidation(),
  [ImportFields.region]: basicStringValidation().regex(
    /^[^\s]*$/,
    'Region cannot contain spaces'
  ),
  [ImportFields.endpoint]: basicStringValidation().url(),
  [ImportFields.accessKey]: basicStringValidation(),
  [ImportFields.secretKey]: basicStringValidation(),
  [ImportFields.verifyTlS]: z.boolean(),
  [ImportFields.forcePathStyle]: z.boolean(),
};
export const s3Schema = z.object(s3SchemaObject);

const filePathSchemaObject = {
  [ImportFields.filePath]: basicStringValidation().regex(
    /^\/?([^/\s]+\/)*[^/\s]*$/,
    'Invalid file path'
  ),
};
export const filePathSchema = z.object(filePathSchemaObject);

export const importStepSchema = z
  .object({
    ...dataImporterSchemeObject,
    ...s3SchemaObject,
    ...filePathSchemaObject,
    showCreds: z.boolean().optional(),
    credentials: z.record(basicStringValidation()).optional(),
  })
  .superRefine((data, ctx) => {
    if (data.showCreds) {
      if (!data.credentials || Object.keys(data.credentials).length === 0) {
        ctx.addIssue({
          path: ['credentials'],
          code: z.ZodIssueCode.custom,
          message: 'Credentials are required',
        });
      }
    }
  });
