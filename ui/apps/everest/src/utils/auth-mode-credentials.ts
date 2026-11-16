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

import {
  Component,
  FieldType,
  Section,
} from 'components/ui-generator/ui-generator.types';

// An authentication mode as monitoring and backup storage classes declare it:
// the keys its credentials Secret must or may carry.
export interface AuthModeCredentials {
  name: string;
  credentials?: { required: string[]; optional?: string[] };
}

const toMaskedCredentialField = (
  key: string,
  required: boolean
): Component => ({
  path: key,
  uiType: FieldType.Text,
  fieldParams: { label: key, type: 'password' },
  ...(required && { validation: { required: true } }),
});

// The class's credentials form for the mode or, when the class ships none, one
// masked input per credential key the mode declares.
export const getAuthModeCredentialsSection = (
  authModes: AuthModeCredentials[],
  credentialsForms: Partial<Record<string, Section>>,
  mode: string
): Section | undefined => {
  const section = credentialsForms[mode];
  if (section) {
    return section;
  }
  const credentials = authModes.find(
    (authMode) => authMode.name === mode
  )?.credentials;
  const required = credentials?.required ?? [];
  const keys = [...required, ...(credentials?.optional ?? [])];
  if (keys.length === 0) {
    return undefined;
  }
  return {
    componentsOrder: keys,
    components: Object.fromEntries(
      keys.map((key) => [
        key,
        toMaskedCredentialField(key, required.includes(key)),
      ])
    ),
  };
};
