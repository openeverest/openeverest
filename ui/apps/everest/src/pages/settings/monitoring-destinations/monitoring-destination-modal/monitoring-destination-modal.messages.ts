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
  replaceCredentials: (secret: string) =>
    `Replace the credentials in Secret ${secret}`,
  credentialsSecretName: 'Credentials Secret name',
  credentialsSecretNameHelper: (defaultName: string) =>
    `The Secret the credentials are stored in. Defaults to ${defaultName}.`,
  secretNameFieldName: 'Secret name',
  secretExists: (secret: string) =>
    `A Secret named ${secret} already exists. Reuse it as is, or enter another Credentials Secret name.`,
  reuseSecret: 'Reuse it',
  changedMeanwhile:
    'This destination was changed meanwhile. The form now shows the latest version; review it and save again.',
  tls: 'TLS',
  skipTlsVerification: 'Skip TLS certificate verification',
  caSecretName: 'CA certificate Secret',
  caSecretKey: 'Key in the Secret',
  caHelper: 'Optional: the CA bundle that signed the server certificate.',
  caIncomplete: 'Set both the Secret and its key, or neither',
};
