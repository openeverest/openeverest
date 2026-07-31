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

import { buildRestoreDataSource } from './restore-data-source';

describe('buildRestoreDataSource', () => {
  it('builds a backup data source with a backupRef', () => {
    expect(
      buildRestoreDataSource({ type: 'Backup', backupName: 'daily-1' })
    ).toEqual({ type: 'Backup', backup: { backupRef: { name: 'daily-1' } } });
  });

  it('builds a date PITR data source with the date and storageRef', () => {
    expect(
      buildRestoreDataSource({
        type: 'PointInTime',
        storageName: 's3-main',
        recoveryTarget: 'date',
        date: '2026-07-29T10:15:00Z',
      })
    ).toEqual({
      type: 'PointInTime',
      pointInTime: {
        recoveryTarget: 'date',
        date: '2026-07-29T10:15:00Z',
        source: { storageRef: { name: 's3-main' } },
      },
    });
  });

  it('omits date entirely for a latest PITR data source', () => {
    const dataSource = buildRestoreDataSource({
      type: 'PointInTime',
      storageName: 's3-main',
      recoveryTarget: 'latest',
    });

    expect(dataSource).toEqual({
      type: 'PointInTime',
      pointInTime: {
        recoveryTarget: 'latest',
        source: { storageRef: { name: 's3-main' } },
      },
    });
    expect('date' in dataSource.pointInTime!).toBe(false);
  });
});
