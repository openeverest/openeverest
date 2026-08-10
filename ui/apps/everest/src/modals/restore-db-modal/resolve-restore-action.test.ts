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
  BackupTypeValues,
  RecoveryTargetValues,
} from './restore-db-modal-schema';
import { RestorePitrStorageOption } from './restore-db-modal.types';
import { resolveRestoreAction } from './resolve-restore-action';

const storage: RestorePitrStorageOption = {
  name: 's3-main',
  window: {
    available: true,
    earliest: new Date('2026-07-29T00:00:00Z'),
    latest: new Date('2026-07-29T12:00:00Z'),
  },
};

const ctx = {
  instanceName: 'inst-1',
  isNewClusterMode: false,
  pitrStorages: [storage],
};

describe('resolveRestoreAction', () => {
  it('fires a backup restore for an in-place backup submission', () => {
    const action = resolveRestoreAction(
      { backupType: BackupTypeValues.fromBackup, backupName: 'daily-1' },
      ctx
    );

    expect(action).toEqual({
      kind: 'restore',
      variables: {
        instanceName: 'inst-1',
        type: 'Backup',
        backupName: 'daily-1',
      },
    });
  });

  it('navigates to the create-new-DB flow in new-cluster mode', () => {
    const action = resolveRestoreAction(
      { backupType: BackupTypeValues.fromBackup, backupName: 'daily-1' },
      { ...ctx, isNewClusterMode: true }
    );

    expect(action).toEqual({
      kind: 'navigate-new',
      dataSource: { type: 'Backup', backupName: 'daily-1' },
    });
  });

  it('carries the backup storage for a clone when known', () => {
    const action = resolveRestoreAction(
      { backupType: BackupTypeValues.fromBackup, backupName: 'daily-1' },
      {
        ...ctx,
        isNewClusterMode: true,
        succeededBackups: [{ name: 'daily-1', storageName: 's3-main' }],
      }
    );

    expect(action).toEqual({
      kind: 'navigate-new',
      dataSource: {
        type: 'Backup',
        backupName: 'daily-1',
        storageName: 's3-main',
      },
    });
  });

  it('carries a latest PITR clone through navigation with the source instance', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        pitrStorage: 's3-main',
        recoveryTarget: RecoveryTargetValues.latest,
      },
      { ...ctx, isNewClusterMode: true }
    );

    expect(action).toEqual({
      kind: 'navigate-new',
      dataSource: {
        type: 'PointInTime',
        storageName: 's3-main',
        recoveryTarget: 'latest',
        sourceInstanceName: 'inst-1',
      },
    });
  });

  it('carries a date PITR clone through navigation with the source instance', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        pitrStorage: 's3-main',
        recoveryTarget: RecoveryTargetValues.date,
        pointInTimeDate: new Date('2026-07-29T06:30:00.500Z'),
      },
      { ...ctx, isNewClusterMode: true }
    );

    expect(action).toEqual({
      kind: 'navigate-new',
      dataSource: {
        type: 'PointInTime',
        storageName: 's3-main',
        recoveryTarget: 'date',
        date: '2026-07-29T06:30:00Z',
        sourceInstanceName: 'inst-1',
      },
    });
  });

  it('builds a latest PITR restore without a date', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        pitrStorage: 's3-main',
        recoveryTarget: RecoveryTargetValues.latest,
      },
      ctx
    );

    expect(action).toEqual({
      kind: 'restore',
      variables: {
        instanceName: 'inst-1',
        type: 'PointInTime',
        storageName: 's3-main',
        recoveryTarget: 'latest',
      },
    });
  });

  it('builds a date PITR restore with a UTC-serialized date', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        pitrStorage: 's3-main',
        recoveryTarget: RecoveryTargetValues.date,
        pointInTimeDate: new Date('2026-07-29T06:30:00.500Z'),
      },
      ctx
    );

    expect(action).toEqual({
      kind: 'restore',
      variables: {
        instanceName: 'inst-1',
        type: 'PointInTime',
        storageName: 's3-main',
        recoveryTarget: 'date',
        date: '2026-07-29T06:30:00Z',
      },
    });
  });

  it('falls back to the first storage when none is selected', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        recoveryTarget: RecoveryTargetValues.latest,
      },
      ctx
    );

    expect(action).toMatchObject({
      kind: 'restore',
      variables: { storageName: 's3-main' },
    });
  });

  it('resolves to none when a date target has no date', () => {
    const action = resolveRestoreAction(
      {
        backupType: BackupTypeValues.fromPitr,
        pitrStorage: 's3-main',
        recoveryTarget: RecoveryTargetValues.date,
      },
      ctx
    );

    expect(action).toEqual({ kind: 'none' });
  });
});
