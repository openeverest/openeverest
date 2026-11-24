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

// Form fields at the root of the form the block is part of.
export enum AuthenticationFormFields {
  authMode = 'authMode',
  credentialsSource = 'credentialsSource',
  credentialsSecretName = 'credentialsSecretName',
  existingSecretName = 'existingSecretName',
  // Root of the class's credentials form: the Secret's keys.
  credentials = 'credentials',
}

// Key the credentials form is prepared under.
export const CREDENTIALS_SECTION_KEY = 'credentials';

export const MAX_SECRET_NAME_LENGTH = 253;
