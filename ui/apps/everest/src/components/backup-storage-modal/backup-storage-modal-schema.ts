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
  authenticationSchemaShape,
  refineAuthentication,
  SecretNamespacePermissions,
} from 'components/class-authentication';
import { buildOpenApiParametersSchema } from 'components/openapi-parameters-fields';
import {
  refineTlsConfig,
  tlsConfigSchemaShape,
} from 'components/tls-config-fields';
import { getSectionFormZodSchema } from 'components/ui-generator/utils/section-form';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import { getBackupStorageClassName } from 'utils/backup-storage-classes';
import { rfc_123_schema } from 'utils/common-validation';
import { StorageFormFields } from './backup-storage-modal.constants';
import { Messages } from './backup-storage-modal.messages';
import {
  getFormMode,
  getStorageClassForms,
} from './backup-storage-modal.utils';

const required = z.string().min(1, Messages.required);

// The class forms are validated at their roots for the class and auth mode
// picked in the form.
export const buildBackupStorageSchema = ({
  classes,
  original,
  secretPermissions,
}: {
  classes: BackupStorageClass[];
  original?: BackupStorageCRD;
  secretPermissions: SecretNamespacePermissions;
}) =>
  z
    .object({
      // An existing storage keeps the name the API took, whatever made it.
      [StorageFormFields.name]: original
        ? z.string()
        : rfc_123_schema({ fieldName: Messages.nameFieldName }),
      [StorageFormFields.namespace]: required,
      [StorageFormFields.className]: required,
      ...authenticationSchemaShape,
      ...tlsConfigSchemaShape,
    })
    .passthrough()
    .superRefine((values, ctx) => {
      const forms = getStorageClassForms(
        classes.find((c) => getBackupStorageClassName(c) === values.className),
        values.authMode,
        getFormMode(original)
      );
      const validateAt = (root: StorageFormFields, schema: z.ZodTypeAny) =>
        schema
          .safeParse(values[root])
          .error?.issues.forEach((issue) =>
            ctx.addIssue({ ...issue, path: [root, ...issue.path] })
          );

      validateAt(
        StorageFormFields.spec,
        getSectionFormZodSchema(forms.location)
      );
      if (forms.storage) {
        validateAt(
          StorageFormFields.spec,
          getSectionFormZodSchema(forms.storage)
        );
      } else {
        validateAt(
          StorageFormFields.fallbackParameters,
          buildOpenApiParametersSchema(forms.parameterFields)
        );
      }
      refineAuthentication(values, ctx, {
        credentialsForm: forms.credentials,
        saved: original?.spec.authentication,
        permissions: secretPermissions,
      });
      refineTlsConfig(values, ctx);
    });
