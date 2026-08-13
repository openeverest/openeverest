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

import { mapRestore } from './map-restore';

describe('mapRestore', () => {
  it('maps a backup restore, surfacing the backup name and full type', () => {
    expect(
      mapRestore({
        metadata: {
          name: 'restore-1',
          creationTimestamp: '2026-07-29T00:00:00Z',
        },
        spec: {
          instanceRef: { name: 'inst-1' },
          dataSource: {
            type: 'Backup',
            backup: { backupRef: { name: 'daily-1' } },
          },
        },
        status: {
          state: 'Succeeded',
          startedAt: '2026-07-29T01:00:00Z',
          completedAt: '2026-07-29T01:05:00Z',
        },
      })
    ).toEqual({
      name: 'restore-1',
      startTime: '2026-07-29T01:00:00Z',
      endTime: '2026-07-29T01:05:00Z',
      state: 'Succeeded',
      type: 'full',
      backupSource: 'daily-1',
    });
  });

  it('maps a PITR restore, surfacing the source storage and pitr type', () => {
    expect(
      mapRestore({
        metadata: {
          name: 'restore-2',
          creationTimestamp: '2026-07-29T00:00:00Z',
        },
        spec: {
          instanceRef: { name: 'inst-1' },
          dataSource: {
            type: 'PointInTime',
            pointInTime: {
              recoveryTarget: 'latest',
              source: { storageRef: { name: 's3-main' } },
            },
          },
        },
      })
    ).toMatchObject({
      name: 'restore-2',
      type: 'pitr',
      backupSource: 's3-main',
    });
  });

  it('falls back to creation time and unknown state when status is absent', () => {
    const restore = mapRestore({
      metadata: {
        name: 'restore-3',
        creationTimestamp: '2026-07-29T00:00:00Z',
      },
      spec: {
        instanceRef: { name: 'inst-1' },
        dataSource: {
          type: 'Backup',
          backup: { backupRef: { name: 'daily-2' } },
        },
      },
    });

    expect(restore.startTime).toBe('2026-07-29T00:00:00Z');
    expect(restore.state).toBe('unknown');
    expect(restore.endTime).toBeUndefined();
  });
});
