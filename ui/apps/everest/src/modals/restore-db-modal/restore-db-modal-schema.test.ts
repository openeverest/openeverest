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
  RestoreDbFields,
  schema,
} from './restore-db-modal-schema';
import { RestorePitrStorageOption } from './restore-db-modal.types';

const availableStorage: RestorePitrStorageOption = {
  name: 's3-main',
  window: {
    available: true,
    earliest: new Date('2026-07-29T00:00:00Z'),
    latest: new Date('2026-07-29T12:00:00Z'),
  },
};

const unavailableStorage: RestorePitrStorageOption = {
  name: 's3-cold',
  window: { available: false, message: 'no successful backup yet' },
};

const issuePaths = (
  result: ReturnType<ReturnType<typeof schema>['safeParse']>
): Array<string | number> =>
  result.success ? [] : result.error.issues.flatMap((issue) => issue.path);

describe('restore-db-modal schema', () => {
  it('requires a backup name when restoring from a backup', () => {
    const result = schema().safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromBackup,
      [RestoreDbFields.backupName]: '',
    });

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain(RestoreDbFields.backupName);
  });

  it('accepts a backup restore with a selected backup', () => {
    const result = schema().safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromBackup,
      [RestoreDbFields.backupName]: 'daily-1',
    });

    expect(result.success).toBe(true);
  });

  it('rejects PITR when no storage is available', () => {
    const result = schema([]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.latest,
    });

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain(RestoreDbFields.pitrStorage);
  });

  it('rejects PITR on a storage without a trustworthy window', () => {
    const result = schema([unavailableStorage]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.pitrStorage]: unavailableStorage.name,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.latest,
    });

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain(RestoreDbFields.pitrStorage);
  });

  it('accepts a latest PITR restore without a date', () => {
    const result = schema([availableStorage]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.pitrStorage]: availableStorage.name,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.latest,
      [RestoreDbFields.pointInTimeDate]: null,
    });

    expect(result.success).toBe(true);
  });

  it('requires a date for a date-targeted PITR restore', () => {
    const result = schema([availableStorage]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.pitrStorage]: availableStorage.name,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.date,
      [RestoreDbFields.pointInTimeDate]: null,
    });

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain(RestoreDbFields.pointInTimeDate);
  });

  it('accepts a date inside the recovery window', () => {
    const result = schema([availableStorage]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.pitrStorage]: availableStorage.name,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.date,
      [RestoreDbFields.pointInTimeDate]: new Date('2026-07-29T06:00:00Z'),
    });

    expect(result.success).toBe(true);
  });

  it('rejects a date outside the recovery window', () => {
    const result = schema([availableStorage]).safeParse({
      [RestoreDbFields.backupType]: BackupTypeValues.fromPitr,
      [RestoreDbFields.pitrStorage]: availableStorage.name,
      [RestoreDbFields.recoveryTarget]: RecoveryTargetValues.date,
      [RestoreDbFields.pointInTimeDate]: new Date('2026-07-29T18:00:00Z'),
    });

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain(RestoreDbFields.pointInTimeDate);
  });
});
