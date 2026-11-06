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

// Keys the class's destination and credentials forms are prepared under.
export const DESTINATION_SECTION_KEY = 'destination';
export const CREDENTIALS_SECTION_KEY = 'credentials';

export enum DestinationFormFields {
  name = 'name',
  namespace = 'namespace',
  className = 'className',
  authMode = 'authMode',
  replaceCredentials = 'replaceCredentials',
  credentialsSecretName = 'credentialsSecretName',
  // Root of the class credentials form: the Secret's keys.
  credentials = 'credentials',
  // Root of the class destination form, whose paths are relative to the
  // destination spec (e.g. parameters.url).
  spec = 'spec',
  // Root of the form generated from destinationParametersSchema when the
  // class ships no destination form.
  fallbackParameters = 'fallbackParameters',
  insecureSkipVerify = 'insecureSkipVerify',
  caSecretName = 'caSecretName',
  caSecretKey = 'caSecretKey',
}

export const MAX_SECRET_NAME_LENGTH = 253;
