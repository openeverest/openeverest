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

import { FieldType, Section } from 'components/ui-generator/ui-generator.types';
import {
  AuthModeCredentials,
  getAuthModeCredentialsSection,
} from './auth-mode-credentials';

const authModes: AuthModeCredentials[] = [
  {
    name: 'AccessKey',
    credentials: {
      required: ['ACCESS_KEY_ID', 'SECRET_ACCESS_KEY'],
      optional: ['SESSION_TOKEN'],
    },
  },
  { name: 'Anonymous' },
];

describe('getAuthModeCredentialsSection', () => {
  it("returns the class's form for the mode when it ships one", () => {
    const form: Section = {
      componentsOrder: ['ACCESS_KEY_ID'],
      components: {
        ACCESS_KEY_ID: {
          path: 'ACCESS_KEY_ID',
          uiType: FieldType.Text,
          fieldParams: { label: 'Access key ID' },
        },
      },
    };

    expect(
      getAuthModeCredentialsSection(authModes, { AccessKey: form }, 'AccessKey')
    ).toBe(form);
  });

  it('renders one masked input per declared key without a form', () => {
    const section = getAuthModeCredentialsSection(authModes, {}, 'AccessKey');

    expect(section?.componentsOrder).toEqual([
      'ACCESS_KEY_ID',
      'SECRET_ACCESS_KEY',
      'SESSION_TOKEN',
    ]);
    expect(section?.components.SECRET_ACCESS_KEY).toEqual({
      path: 'SECRET_ACCESS_KEY',
      uiType: FieldType.Text,
      fieldParams: { label: 'SECRET_ACCESS_KEY', type: 'password' },
      validation: { required: true },
    });
    expect(section?.components.SESSION_TOKEN).not.toHaveProperty('validation');
  });

  it('returns nothing for a mode without credentials or an unknown mode', () => {
    expect(getAuthModeCredentialsSection(authModes, {}, 'Anonymous')).toBe(
      undefined
    );
    expect(getAuthModeCredentialsSection(authModes, {}, 'BasicAuth')).toBe(
      undefined
    );
  });
});
