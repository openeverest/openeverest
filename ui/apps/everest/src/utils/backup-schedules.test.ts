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

import { Instance } from 'shared-types/api.types';
import {
  flattenSchedules,
  retentionCopiesFromApi,
  retentionToApi,
} from './backup-schedules';

describe('retentionCopiesFromApi', () => {
  it('maps count retention to copies', () => {
    expect(retentionCopiesFromApi({ type: 'count', count: 7 })).toBe(7);
  });

  it('treats unset and time retention as keep-all', () => {
    expect(retentionCopiesFromApi(undefined)).toBeUndefined();
    expect(
      retentionCopiesFromApi({ type: 'time', duration: '30d' })
    ).toBeUndefined();
  });
});

describe('retentionToApi', () => {
  it('maps positive copies to count retention', () => {
    expect(retentionToApi(3)).toEqual({ type: 'count', count: 3 });
  });

  it('omits retention for keep-all (0 / unset)', () => {
    expect(retentionToApi(0)).toBeUndefined();
    expect(retentionToApi(undefined)).toBeUndefined();
  });
});

describe('flattenSchedules', () => {
  it('projects nested retention onto retentionCopies', () => {
    const instance = {
      spec: {
        backup: {
          storages: [
            {
              storageRef: { name: 's3' },
              schedules: [
                {
                  name: 'daily',
                  cron: '0 2 * * *',
                  enabled: true,
                  retention: { type: 'count', count: 2 },
                },
              ],
            },
          ],
        },
      },
    } as unknown as Instance;

    expect(flattenSchedules(instance)).toEqual([
      {
        name: 'daily',
        cron: '0 2 * * *',
        enabled: true,
        retentionCopies: 2,
        parameters: undefined,
        storageName: 's3',
      },
    ]);
  });
});
