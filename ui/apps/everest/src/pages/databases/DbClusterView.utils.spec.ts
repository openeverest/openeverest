// everest
// Copyright (C) 2023 Percona LLC
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

import { describe, it, expect } from 'vitest';
import { getLastBackupStatus, sortBackupsByTime } from './DbClusterView.utils';
import { Backup, BackupStatus } from 'shared-types/backups.types';
import { Schedule } from 'shared-types/dbCluster.types';

const makeBackup = (state: BackupStatus, completedIso?: string): Backup => ({
  name: 'backup-test',
  state,
  dbClusterName: 'test-cluster',
  backupStorageName: 'test-storage',
  ...(completedIso ? { completed: completedIso } : {}),
});

const makeSchedule = (): Schedule => ({
  enabled: true,
  name: 'daily',
  backupStorageName: 'test-storage',
  schedule: '0 0 * * *',
});

describe('getLastBackupStatus', () => {
  it('returns Inactive when no backups and no schedules', () => {
    expect(getLastBackupStatus([], [])).toBe('Inactive');
  });

  it('returns Scheduled when no backups but schedules exist', () => {
    expect(getLastBackupStatus([], [makeSchedule()])).toBe('Scheduled');
  });

  it('returns Pending when a backup is IN_PROGRESS (the previously broken case)', () => {
    const backups = sortBackupsByTime([makeBackup(BackupStatus.IN_PROGRESS)]);
    expect(getLastBackupStatus(backups, [])).toBe('Pending');
  });

  it('returns Pending (not Scheduled) when IN_PROGRESS backup exists alongside schedules', () => {
    const backups = sortBackupsByTime([makeBackup(BackupStatus.IN_PROGRESS)]);
    expect(getLastBackupStatus(backups, [makeSchedule()])).toBe('Pending');
  });

  it('returns Inactive when all backups are FAILED and no schedules', () => {
    const backups = sortBackupsByTime([
      makeBackup(BackupStatus.FAILED, '2024-01-01T00:00:00Z'),
      makeBackup(BackupStatus.FAILED, '2024-01-02T00:00:00Z'),
    ]);
    expect(getLastBackupStatus(backups, [])).toBe('Inactive');
  });

  it('returns Scheduled when all backups are FAILED but schedules exist', () => {
    const backups = sortBackupsByTime([
      makeBackup(BackupStatus.FAILED, '2024-01-01T00:00:00Z'),
    ]);
    expect(getLastBackupStatus(backups, [makeSchedule()])).toBe('Scheduled');
  });

  it('returns Not Started for UNKNOWN state', () => {
    const backups = sortBackupsByTime([makeBackup(BackupStatus.UNKNOWN)]);
    expect(getLastBackupStatus(backups, [])).toBe('Not Started');
  });

  it('returns Pending when latest backup is IN_PROGRESS and an older OK backup exists', () => {
    // OK backup has an older completed date; IN_PROGRESS has no completed → sorts last
    const backups = sortBackupsByTime([
      makeBackup(BackupStatus.OK, '2024-01-01T00:00:00Z'),
      makeBackup(BackupStatus.IN_PROGRESS),
    ]);
    expect(getLastBackupStatus(backups, [])).toBe('Pending');
  });

  it('returns Pending when latest backup is IN_PROGRESS and an older FAILED backup exists', () => {
    const backups = sortBackupsByTime([
      makeBackup(BackupStatus.FAILED, '2024-01-01T00:00:00Z'),
      makeBackup(BackupStatus.IN_PROGRESS),
    ]);
    expect(getLastBackupStatus(backups, [])).toBe('Pending');
  });
});
