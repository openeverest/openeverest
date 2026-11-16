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
import {
  getSectionFormZodSchema,
  SectionForm,
} from 'components/ui-generator/utils/section-form';
import { rfc_123_schema } from 'utils/common-validation';
import {
  AuthenticationFormFields,
  MAX_SECRET_NAME_LENGTH,
} from './class-authentication.constants';
import { Messages } from './class-authentication.messages';
import {
  AuthenticationFormValues,
  CredentialsSource,
  SavedAuthentication,
  SecretNamespacePermissions,
} from './class-authentication.types';
import {
  getCredentialsSource,
  takesCredentials,
} from './class-authentication.utils';

export const authenticationSchemaShape = {
  [AuthenticationFormFields.authMode]: z.string().min(1, Messages.required),
  [AuthenticationFormFields.credentialsSource]: z.nativeEnum(CredentialsSource),
  [AuthenticationFormFields.credentialsSecretName]: z.union([
    z.literal(''),
    rfc_123_schema({
      fieldName: Messages.secretNameFieldName,
      maxLength: MAX_SECRET_NAME_LENGTH,
    }),
  ]),
  [AuthenticationFormFields.existingSecretName]: z.string(),
};

// What the picked credentials source needs, checked only for a mode that
// keeps its credentials in a Secret (one with a credentials form).
export const refineAuthentication = (
  values: Pick<
    AuthenticationFormValues,
    'authMode' | 'credentialsSource' | 'existingSecretName'
  > & { namespace: string; credentials?: unknown },
  ctx: z.RefinementCtx,
  {
    credentialsForm,
    saved,
    permissions,
  }: {
    credentialsForm?: SectionForm;
    saved: SavedAuthentication | undefined;
    permissions: SecretNamespacePermissions;
  }
) => {
  if (!credentialsForm) {
    return;
  }
  const source = getCredentialsSource(values, saved, permissions);
  if (!source) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: [AuthenticationFormFields.credentialsSource],
      message: Messages.noSecretAccess(values.namespace),
    });
  } else if (takesCredentials(source)) {
    getSectionFormZodSchema(credentialsForm)
      .safeParse(values.credentials)
      .error?.issues.forEach((issue) =>
        ctx.addIssue({
          ...issue,
          path: [AuthenticationFormFields.credentials, ...issue.path],
        })
      );
  } else if (
    source === CredentialsSource.Existing &&
    !values.existingSecretName
  ) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: [AuthenticationFormFields.existingSecretName],
      message: Messages.required,
    });
  }
};
