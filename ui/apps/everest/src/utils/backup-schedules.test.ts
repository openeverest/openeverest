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
import { applySchedulesToStorages } from 'pages/db-cluster-details/backups/backups.utils';
import { flattenSchedules, scheduleToApi } from './backup-schedules';

describe('scheduleToApi', () => {
  it('omits unset retention', () => {
    expect(
      scheduleToApi({
        name: 'daily',
        cron: '0 2 * * *',
        enabled: true,
        storageName: 's3',
      })
    ).toEqual({
      name: 'daily',
      cron: '0 2 * * *',
      enabled: true,
    });
  });
});

describe('flattenSchedules', () => {
  it('passes nested retention through onto FlattenedSchedule', () => {
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
        retention: { type: 'count', count: 2 },
        storageName: 's3',
      },
    ]);
  });
});

describe('applySchedulesToStorages', () => {
  it('round-trips a time retention schedule unchanged', () => {
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
                  retention: { type: 'time', duration: '30d' },
                },
              ],
            },
          ],
        },
      },
    } as unknown as Instance;

    const schedules = flattenSchedules(instance);
    const storages = applySchedulesToStorages(instance, schedules);

    expect(storages[0].schedules).toEqual([
      {
        name: 'daily',
        cron: '0 2 * * *',
        enabled: true,
        retention: { type: 'time', duration: '30d' },
      },
    ]);
  });
});
