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

import { CredentialsSource } from './monitoring-destination-modal.types';

const EXISTING_SECRET_OPTION = 'Use an existing Secret';

export const Messages = {
  createTitle: 'Add monitoring destination',
  editTitle: 'Edit monitoring destination',
  create: 'Add',
  save: 'Save',
  name: 'Name',
  nameFieldName: 'name',
  namespace: 'Namespace',
  monitoringClass: 'Type',
  authMode: 'Authentication',
  required: 'Required',
  credentials: 'Credentials',
  skippedParameters: (names: string[]) =>
    `Not editable here: ${names.join(', ')}. Set them through the API.`,
  credentialsSource: {
    [CredentialsSource.Keep]: (secret: string) => `Keep ${secret}`,
    [CredentialsSource.Update]: () => 'Enter new credentials',
    [CredentialsSource.New]: () => 'Enter credentials',
    [CredentialsSource.Existing]: () => EXISTING_SECRET_OPTION,
  } satisfies Record<CredentialsSource, (secret: string) => string>,
  updateCredentialsCaption: (secret: string) =>
    `They replace the credentials in Secret ${secret}, also for any other destination that uses it.`,
  keptCredentials: (secret: string) => `Kept in Secret ${secret}.`,
  credentialsSecretName: 'Credentials Secret name',
  credentialsSecretNameHelper: (defaultName: string) =>
    `The Secret the credentials are stored in. Defaults to ${defaultName}.`,
  secretNameFieldName: 'Secret name',
  existingSecret: 'Secret',
  existingSecretHelper:
    'Secrets that hold credentials of this type and authentication mode.',
  loadingSecrets: 'Loading Secrets...',
  secretsLoadFailed: 'Secrets could not be loaded. Try again later.',
  noExistingSecrets: (namespace: string) =>
    `No Secrets with credentials of this type and authentication mode in the ${namespace} namespace yet.`,
  noSecretAccess: (namespace: string) =>
    `This authentication mode keeps its credentials in a Secret, and you don't have permission to use Secrets in the ${namespace} namespace.`,
  secretExists: (secret: string) =>
    `A Secret named ${secret} already exists. Pick it under "${EXISTING_SECRET_OPTION}", or enter another Credentials Secret name.`,
  secretTaken: (secret: string) =>
    `A Secret named ${secret} already exists and doesn't hold credentials for this monitoring class and authentication mode. Enter another Credentials Secret name.`,
  secretChangedMeanwhile: (secret: string) =>
    `Secret ${secret} changed while saving. Save again to store the new credentials.`,
  changedMeanwhile:
    'This destination was changed meanwhile. The form now shows the latest version with your changes kept; review it and save again.',
  restartsInstances: (count: number) => {
    const instances =
      count === 1
        ? 'the instance that sends to this destination'
        : `the ${count} instances that send to this destination`;
    return `Saving these changes may restart the pods of ${instances}, one at a time; an instance with a single replica may be briefly unavailable.`;
  },
  tls: 'TLS',
  skipTlsVerification: 'Skip TLS certificate verification',
  caSecretName: 'CA certificate Secret',
  caSecretKey: 'Key in the Secret',
  caHelper: 'Optional: the CA bundle that signed the server certificate.',
  caIncomplete: 'Set both the Secret and its key, or neither',
};
