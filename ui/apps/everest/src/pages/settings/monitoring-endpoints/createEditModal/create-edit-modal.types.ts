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
import { MonitoringConfig } from 'shared-types/api.types';
import { rfc_123_schema } from 'utils/common-validation';
import { Messages } from '../monitoring-endpoints.messages';

export enum EndpointFormFields {
  name = 'name',
  namespace = 'namespace',
  url = 'url',
  user = 'user',
  password = 'password',
  verifyTLS = 'verifyTLS',
}

export interface CreateEditEndpointModalProps {
  open: boolean;
  handleClose: () => void;
  handleSubmit: (isEditMode: boolean, data: EndpointFormType) => void;
  selectedEndpoint?: MonitoringConfig;
  selectedNamespace?: string;
  isLoading?: boolean;
}

export const getEndpointSchema = (isEditMode: boolean) =>
  z
    .object({
      [EndpointFormFields.name]: rfc_123_schema({
        fieldName: 'endpoint name',
        maxLength: 22,
      }),
      [EndpointFormFields.namespace]: z.string().nonempty(),
      [EndpointFormFields.verifyTLS]: z.boolean(),
      [EndpointFormFields.url]: z.string().min(1).url(),
      // Credentials are stored in a K8s Secret and are not returned by the GET API.
      // In create mode they are required; in edit mode they are optional (leave empty to keep existing).
      ...(isEditMode
        ? {
            [EndpointFormFields.user]: z.string(),
            [EndpointFormFields.password]: z.string(),
          }
        : {
            [EndpointFormFields.user]: z.string().min(1),
            [EndpointFormFields.password]: z.string().min(1),
          }),
    })
    .superRefine((arg, ctx) => {
      const hasUser = !!arg.user;
      const hasPassword = !!arg.password;

      if (hasUser !== hasPassword) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [
            arg.user ? EndpointFormFields.password : EndpointFormFields.user,
          ],
          message: Messages.helperText.credentials,
        });
      }
    });

export const endpointDefaultValues = {
  [EndpointFormFields.name]: '',
  [EndpointFormFields.namespace]: '',
  [EndpointFormFields.url]: '',
  [EndpointFormFields.user]: '',
  [EndpointFormFields.password]: '',
  [EndpointFormFields.verifyTLS]: true,
};

export type EndpointFormType = z.infer<ReturnType<typeof getEndpointSchema>>;
