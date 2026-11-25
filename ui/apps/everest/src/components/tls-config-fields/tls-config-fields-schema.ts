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
import { TlsConfigFormFields } from './tls-config-fields.constants';
import { Messages } from './tls-config-fields.messages';
import { TlsConfigFormValues } from './tls-config-fields.types';

export const tlsConfigSchemaShape = {
  [TlsConfigFormFields.insecureSkipVerify]: z.boolean(),
  [TlsConfigFormFields.caSecretName]: z.string(),
  [TlsConfigFormFields.caSecretKey]: z.string(),
};

// A CA bundle needs both the Secret and its key; skipping verification needs
// neither.
export const refineTlsConfig = (
  { insecureSkipVerify, caSecretName, caSecretKey }: TlsConfigFormValues,
  ctx: z.RefinementCtx
) => {
  if (!insecureSkipVerify && !caSecretName !== !caSecretKey) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: [
        caSecretName
          ? TlsConfigFormFields.caSecretKey
          : TlsConfigFormFields.caSecretName,
      ],
      message: Messages.caIncomplete,
    });
  }
};
