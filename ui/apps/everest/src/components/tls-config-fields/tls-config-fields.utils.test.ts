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

import { getTlsConfigFormValues, toTlsConfig } from './tls-config-fields.utils';

describe('toTlsConfig', () => {
  it('sends skipping verification alone, and a CA only with both its Secret and key', () => {
    expect(
      toTlsConfig({
        insecureSkipVerify: true,
        caSecretName: 'ca',
        caSecretKey: 'ca.crt',
      })
    ).toEqual({ insecureSkipVerify: true });
    expect(
      toTlsConfig({
        insecureSkipVerify: false,
        caSecretName: 'ca',
        caSecretKey: '',
      })
    ).toBeUndefined();
  });

  it.each([
    { insecureSkipVerify: true },
    { ca: { secretKeyRef: { name: 'ca', key: 'ca.crt' } } },
  ])('sends a saved tls back as it was: %j', (tls) => {
    expect(toTlsConfig(getTlsConfigFormValues(tls))).toEqual(tls);
  });
});
