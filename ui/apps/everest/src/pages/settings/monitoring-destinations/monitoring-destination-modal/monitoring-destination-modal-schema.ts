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
import { buildOpenApiParametersSchema } from 'components/openapi-parameters-fields';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { rfc_123_schema } from 'utils/common-validation';
import {
  getMonitoringClassName,
  getSectionFormZodSchema,
} from 'utils/monitoring-classes';
import {
  DestinationFormFields,
  MAX_SECRET_NAME_LENGTH,
} from './monitoring-destination-modal.constants';
import { Messages } from './monitoring-destination-modal.messages';
import {
  getCredentialsSecretAction,
  getDestinationClassForms,
  getFormMode,
} from './monitoring-destination-modal.utils';

const required = z.string().min(1, Messages.required);

// The class forms are validated at their roots for the class and auth mode
// picked in the form.
export const buildMonitoringDestinationSchema = ({
  classes,
  original,
}: {
  classes: MonitoringClass[];
  original?: MonitoringDestination;
}) =>
  z
    .object({
      [DestinationFormFields.name]: rfc_123_schema({
        fieldName: Messages.nameFieldName,
      }),
      [DestinationFormFields.namespace]: required,
      [DestinationFormFields.className]: required,
      [DestinationFormFields.authMode]: required,
      [DestinationFormFields.replaceCredentials]: z.boolean(),
      [DestinationFormFields.credentialsSecretName]: z.union([
        z.literal(''),
        rfc_123_schema({
          fieldName: Messages.secretNameFieldName,
          maxLength: MAX_SECRET_NAME_LENGTH,
        }),
      ]),
      [DestinationFormFields.insecureSkipVerify]: z.boolean(),
      [DestinationFormFields.caSecretName]: z.string(),
      [DestinationFormFields.caSecretKey]: z.string(),
    })
    .passthrough()
    .superRefine((values, ctx) => {
      const formMode = getFormMode(original);
      const forms = getDestinationClassForms(
        classes.find((c) => getMonitoringClassName(c) === values.className),
        values.authMode,
        formMode
      );
      const validateAt = (root: DestinationFormFields, schema: z.ZodTypeAny) =>
        schema
          .safeParse(values[root])
          .error?.issues.forEach((issue) =>
            ctx.addIssue({ ...issue, path: [root, ...issue.path] })
          );

      if (forms.destination) {
        validateAt(
          DestinationFormFields.spec,
          getSectionFormZodSchema(forms.destination, formMode)
        );
      } else {
        validateAt(
          DestinationFormFields.fallbackParameters,
          buildOpenApiParametersSchema(forms.parameterFields)
        );
      }
      if (forms.credentials && getCredentialsSecretAction(values, original)) {
        validateAt(
          DestinationFormFields.credentials,
          getSectionFormZodSchema(forms.credentials)
        );
      }
      if (
        !values.insecureSkipVerify &&
        !values.caSecretName !== !values.caSecretKey
      ) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [
            values.caSecretName
              ? DestinationFormFields.caSecretKey
              : DestinationFormFields.caSecretName,
          ],
          message: Messages.caIncomplete,
        });
      }
    });
