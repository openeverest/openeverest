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

import { InstanceBackupStorage } from './backups.types';
import {
  buildPitrPayload,
  countPitrEnabledStorages,
  getPitrBlockReason,
  setStoragePitr,
} from './pitr.utils';

const storages: InstanceBackupStorage[] = [
  {
    storageRef: { name: 's3-prod' },
    schedules: [{ name: 'daily', cron: '0 2 * * *', enabled: true }],
    pitr: { enabled: true },
  },
  {
    storageRef: { name: 's3-archive' },
    schedules: [{ name: 'weekly', cron: '0 3 * * 0', enabled: false }],
  },
];

describe('setStoragePitr', () => {
  it('updates PITR only on the matching storage', () => {
    const result = setStoragePitr(storages, 's3-archive', { enabled: true });

    expect(result[1].pitr).toEqual({ enabled: true });
    expect(result[0]).toBe(storages[0]);
  });

  it('preserves the rest of the matched storage', () => {
    const result = setStoragePitr(storages, 's3-prod', { enabled: false });

    expect(result[0].pitr).toEqual({ enabled: false });
    expect(result[0].schedules).toBe(storages[0].schedules);
  });

  it('returns an unchanged copy when no storage name matches', () => {
    const result = setStoragePitr(storages, 'missing', { enabled: true });

    expect(result).toEqual(storages);
  });
});

describe('countPitrEnabledStorages', () => {
  it('counts only storages with PITR enabled', () => {
    expect(countPitrEnabledStorages(storages)).toBe(1);
  });
});

describe('getPitrBlockReason', () => {
  it('never blocks a storage that is already enabled', () => {
    expect(
      getPitrBlockReason({
        enabledCount: 5,
        maxEnabled: 1,
        storageEnabled: true,
      })
    ).toBeUndefined();
  });

  it('blocks with limit-reached when the enabled count meets the limit', () => {
    expect(
      getPitrBlockReason({
        enabledCount: 1,
        maxEnabled: 1,
        storageEnabled: false,
      })
    ).toBe('limit-reached');
  });

  it('allows enabling when under the limit', () => {
    expect(
      getPitrBlockReason({
        enabledCount: 1,
        maxEnabled: 2,
        storageEnabled: false,
      })
    ).toBeUndefined();
  });

  it('allows enabling when the limit is undefined', () => {
    expect(
      getPitrBlockReason({
        enabledCount: 99,
        maxEnabled: undefined,
        storageEnabled: false,
      })
    ).toBeUndefined();
  });

  it('does not block when there is no active schedule', () => {
    expect(
      getPitrBlockReason({
        enabledCount: 0,
        maxEnabled: 5,
        storageEnabled: false,
      })
    ).toBeUndefined();
  });
});

describe('buildPitrPayload', () => {
  it('flips enabled while preserving existing parameters', () => {
    const result = buildPitrPayload(
      { enabled: false, parameters: { timeBetweenUploads: 60 } },
      { enabled: true }
    );

    expect(result).toEqual({
      enabled: true,
      parameters: { timeBetweenUploads: 60 },
    });
  });

  it('overwrites parameters when the patch provides them', () => {
    const result = buildPitrPayload(
      { enabled: true, parameters: { timeBetweenUploads: 60 } },
      { enabled: true, parameters: { timeBetweenUploads: 120 } }
    );

    expect(result).toEqual({
      enabled: true,
      parameters: { timeBetweenUploads: 120 },
    });
  });

  it('builds a payload from no previous pitr', () => {
    expect(buildPitrPayload(undefined, { enabled: true })).toEqual({
      enabled: true,
    });
  });
});
