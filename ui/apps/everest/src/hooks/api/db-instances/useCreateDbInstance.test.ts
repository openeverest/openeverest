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

import { describe, expect, it } from 'vitest';
import { buildCreateInstanceSpec } from './useCreateDbInstance';
import type { DbWizardType } from 'pages/database-form/database-form-schema';

describe('buildCreateInstanceSpec', () => {
  it('preserves topology.type when spec.topology.config is present', () => {
    const formValue = {
      provider: 'percona-server-mongodb',
      dbName: 'test-db',
      k8sNamespace: 'ns',
      topology: { type: 'sharded' },
      spec: {
        topology: {
          config: {
            shards: 1,
          },
        },
      },
    } as unknown as DbWizardType;

    const result = buildCreateInstanceSpec(formValue);

    expect(result).toMatchObject({
      provider: 'percona-server-mongodb',
      topology: {
        type: 'sharded',
        config: {
          shards: 1,
        },
      },
    });
  });
});
