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
  getAvailableStorages,
  GetAvailableStoragesParams,
} from './backup-storages-input.utils';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';

describe('getAvailableStorages', () => {
  const mkStorage = (name: string) =>
    ({
      metadata: { name },
      spec: { type: 's3' },
    }) as unknown as BackupStorageCRD;

  const allStorages: BackupStorageCRD[] = [
    mkStorage('storage1'),
    mkStorage('storage2'),
    mkStorage('storage3'),
    mkStorage('storage4'),
  ];

  const baseParams: GetAvailableStoragesParams = {
    backupStorages: allStorages,
    schedules: [],
  };

  describe('when maxStorages is undefined (no limit)', () => {
    it('shows all namespace storages', () => {
      const result = getAvailableStorages({
        ...baseParams,
        instanceStorageNames: ['storage1'],
      });
      expect(result.storagesToShow).toEqual(allStorages);
      expect(result.limitReached).toBe(false);
      expect(result.shouldDisable).toBe(false);
      expect(result.activeStoragesCount).toBe(1);
    });
  });

  describe('when limit > active storages', () => {
    it('shows all namespace storages', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 2,
        instanceStorageNames: ['storage1'],
      });
      expect(result.storagesToShow.map((s) => s.metadata?.name)).toEqual([
        'storage1',
        'storage2',
        'storage3',
        'storage4',
      ]);
      expect(result.limitReached).toBe(false);
      expect(result.shouldDisable).toBe(false);
    });

    it('provides inUseNames for highlighting', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 3,
        instanceStorageNames: ['storage1', 'storage2'],
      });
      expect(result.inUseNames).toEqual(new Set(['storage1', 'storage2']));
    });
  });

  describe('when active == 0 (no storages on instance yet)', () => {
    it('shows all namespace storages regardless of limit', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 1,
        instanceStorageNames: [],
      });
      expect(result.storagesToShow).toEqual(allStorages);
      expect(result.limitReached).toBe(false);
      expect(result.shouldDisable).toBe(false);
      expect(result.activeStoragesCount).toBe(0);
    });
  });

  describe('when limit == active && limit > 1', () => {
    it('shows only instance storages', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 2,
        instanceStorageNames: ['storage1', 'storage3'],
      });
      expect(result.storagesToShow.map((s) => s.metadata?.name)).toEqual([
        'storage1',
        'storage3',
      ]);
      expect(result.limitReached).toBe(true);
      expect(result.shouldDisable).toBe(false);
    });
  });

  describe('when limit == active == 1', () => {
    it('shows single instance storage and shouldDisable is true', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 1,
        instanceStorageNames: ['storage2'],
      });
      expect(result.storagesToShow.map((s) => s.metadata?.name)).toEqual([
        'storage2',
      ]);
      expect(result.limitReached).toBe(true);
      expect(result.shouldDisable).toBe(true);
      expect(result.activeStoragesCount).toBe(1);
    });
  });

  describe('cascading maxSchedulesPerStorage filter', () => {
    it('removes storages that have reached schedule limit', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 2,
        maxSchedulesPerStorage: 1,
        instanceStorageNames: ['storage1', 'storage2'],
        schedules: [
          {
            backupStorageName: 'storage1',
          },
        ],
      });
      // storage1 has 1 schedule, maxSchedulesPerStorage is 1 → filtered out
      expect(result.storagesToShow.map((s) => s.metadata?.name)).toEqual([
        'storage2',
      ]);
    });

    it('applies after maxStorages filter', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 3,
        maxSchedulesPerStorage: 2,
        instanceStorageNames: ['storage1', 'storage2', 'storage3'],
        schedules: [
          {
            backupStorageName: 'storage1',
          },
          {
            backupStorageName: 'storage1',
          },
          {
            backupStorageName: 'storage2',
          },
        ],
      });
      // limit reached (3==3) → only instance storages shown
      // storage1 has 2 schedules (>= maxSchedulesPerStorage 2) → filtered
      // storage2 has 1 schedule (< 2) → kept
      // storage3 has 0 schedules → kept
      expect(result.storagesToShow.map((s) => s.metadata?.name)).toEqual([
        'storage2',
        'storage3',
      ]);
    });
  });

  describe('backward compatibility (no instanceStorageNames)', () => {
    it('falls back to 0 active count when instanceStorageNames not provided', () => {
      const result = getAvailableStorages({
        ...baseParams,
        maxStorages: 1,
      });
      // No instanceStorageNames → activeStoragesCount = 0 → shows all
      expect(result.storagesToShow).toEqual(allStorages);
      expect(result.activeStoragesCount).toBe(0);
      expect(result.limitReached).toBe(false);
    });
  });
});
