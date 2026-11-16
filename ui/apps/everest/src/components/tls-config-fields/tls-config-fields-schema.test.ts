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
  refineTlsConfig,
  tlsConfigSchemaShape,
} from './tls-config-fields-schema';
import { Messages } from './tls-config-fields.messages';
import { TlsConfigFormValues } from './tls-config-fields.types';

const schema = z.object(tlsConfigSchemaShape).superRefine(refineTlsConfig);
const issuesOf = (values: TlsConfigFormValues) =>
  (schema.safeParse(values).error?.issues ?? []).map(({ path, message }) => ({
    path: path.join('.'),
    message,
  }));

describe('refineTlsConfig', () => {
  it('asks for the missing half of a CA bundle, on the field left empty', () => {
    expect(
      issuesOf({
        insecureSkipVerify: false,
        caSecretName: 'ca',
        caSecretKey: '',
      })
    ).toEqual([{ path: 'caSecretKey', message: Messages.caIncomplete }]);
    expect(
      issuesOf({
        insecureSkipVerify: false,
        caSecretName: '',
        caSecretKey: 'ca.crt',
      })
    ).toEqual([{ path: 'caSecretName', message: Messages.caIncomplete }]);
  });

  it('takes no CA, or none at all when verification is skipped', () => {
    expect(
      issuesOf({ insecureSkipVerify: false, caSecretName: '', caSecretKey: '' })
    ).toEqual([]);
    expect(
      issuesOf({
        insecureSkipVerify: true,
        caSecretName: 'ca',
        caSecretKey: '',
      })
    ).toEqual([]);
  });
});
