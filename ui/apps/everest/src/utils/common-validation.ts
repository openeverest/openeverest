import { capitalize } from '@mui/material';
import z, { IssueData } from 'zod';

const tooLongErrorMessage = (fieldName: string) =>
  `The ${fieldName} name is too long`;

export const errorMessages = {
  tooLong: (value: string) => tooLongErrorMessage(value),
  doesNotEndWithDash: "The name shouldn't end with a hyphen.",
  doesNotStartWithDash: "The name shouldn't start with a hyphen or a number.",
};

export const doesNotContainerAnythingButAlphanumericAndDash = /^[a-z0-9-]+$/;
export const doesNotStartWithDash = /^[^0-9-]/;
export const doesNotEndWithDash = /[^-]$/;

// TODO split message for exceed number message and alphanumeric symbols
// TODO check messages with Catalina

export const rfc_123_schema = ({
  fieldName,
  maxLength = 63,
}: {
  fieldName: string;
  maxLength?: number;
}) =>
  z
    .string()
    .min(1)
    .max(maxLength)
    .regex(
      doesNotContainerAnythingButAlphanumericAndDash,
      `The ${fieldName} should only contain lowercase letters, numbers and hyphens.`
    )
    .regex(doesNotEndWithDash, errorMessages.doesNotEndWithDash)
    .regex(doesNotStartWithDash, errorMessages.doesNotStartWithDash)
    .trim();

type PerconaCustomZodIssueOptions = Partial<Pick<IssueData, 'code'>> &
  Omit<IssueData, 'code'>;

const getPerconaZodCustomIssuePath = (field: string | string[]) =>
  Array.isArray(field) ? field : [field];

export const PerconaZodCustomIssue = {
  required: (
    field: string,
    fieldLabel: string = capitalize(field),
    options?: PerconaCustomZodIssueOptions
  ): IssueData => ({
    path: getPerconaZodCustomIssuePath(field),
    message: `${fieldLabel} is required`,
    ...options,
    code: z.ZodIssueCode.custom,
  }),
  between: (
    field: string,
    min: number,
    max: number,
    fieldLabel: string = field,
    options?: PerconaCustomZodIssueOptions
  ): IssueData => ({
    path: getPerconaZodCustomIssuePath(field),
    message: `${fieldLabel} must be between ${min} and ${max}`,
    ...options,
    code: z.ZodIssueCode.custom,
  }),
};

export const fileValidation = (
  file: File,
  ctx: z.RefinementCtx,
  maxSize: number = 5000000
) => {
  if (!file.name || file.size === 0) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'File is required',
    });
  }

  if (file.size > maxSize) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'File is too large',
    });
  }
};
