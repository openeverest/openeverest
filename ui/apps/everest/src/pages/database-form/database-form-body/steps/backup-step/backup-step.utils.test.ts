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

import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { Instance } from 'shared-types/api.types';
import { WizardBackupSpec } from './backup-step.types';
import {
  buildBackupSpecFromWizard,
  extractWizardBackup,
  ensureStorageRegistered,
} from './backup-step.utils';

const schedule = (
  storageName: string,
  overrides: Partial<FlattenedSchedule> = {}
): FlattenedSchedule => ({
  name: `sch-${storageName}`,
  cron: '0 2 * * *',
  enabled: true,
  storageName,
  ...overrides,
});

const storagesOf = (spec: WizardBackupSpec | undefined) => spec?.storages ?? [];

describe('buildBackupSpecFromWizard', () => {
  it('returns undefined without schedules or a class', () => {
    expect(buildBackupSpecFromWizard([], 'cls')).toBeUndefined();
    expect(
      buildBackupSpecFromWizard([schedule('s3')], undefined)
    ).toBeUndefined();
  });

  it('attaches enabled PITR to the matching storage', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: true },
    });

    expect(storagesOf(spec)[0].pitr).toEqual({ enabled: true });
  });

  it('includes parameters when the entry has them', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: true, parameters: { timeBetweenUploads: 60 } },
    });

    expect(storagesOf(spec)[0].pitr).toEqual({
      enabled: true,
      parameters: { timeBetweenUploads: 60 },
    });
  });

  it('omits pitr for a disabled entry', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: false },
    });

    expect(storagesOf(spec)[0].pitr).toBeUndefined();
  });

  it('ignores an orphan pitr entry for a storage without schedules', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      removed: { enabled: true },
    });
    const storages = storagesOf(spec);

    expect(storages).toHaveLength(1);
    expect(storages[0].name).toBe('s3');
    expect(storages[0].pitr).toBeUndefined();
  });
});

describe('extractWizardBackup', () => {
  const instance = {
    spec: {
      backup: {
        classRef: { name: 'pxc' },
        enabled: true,
        storages: [
          {
            storageRef: { name: 's3-a' },
            schedules: [
              {
                name: 'daily',
                cron: '0 2 * * *',
                enabled: true,
                retentionCopies: 2,
              },
            ],
            pitr: { enabled: true },
          },
          {
            storageRef: { name: 's3-b' },
            schedules: [{ name: 'hourly', cron: '0 * * * *', enabled: false }],
          },
        ],
      },
    },
  } as unknown as Instance;

  it('flattens schedules, keeps the class, and maps enabled PITR per storage', () => {
    expect(extractWizardBackup(instance)).toEqual({
      classRef: { name: 'pxc' },
      schedules: [
        {
          name: 'daily',
          cron: '0 2 * * *',
          enabled: true,
          retentionCopies: 2,
          storageName: 's3-a',
        },
        {
          name: 'hourly',
          cron: '0 * * * *',
          enabled: false,
          storageName: 's3-b',
        },
      ],
      pitr: { 's3-a': { enabled: true } },
    });
  });

  it('returns empty wizard backup for an instance without a backup spec', () => {
    expect(extractWizardBackup({ spec: {} } as unknown as Instance)).toEqual({
      classRef: { name: '' },
      schedules: [],
      pitr: {},
    });
  });
});

describe('ensureStorageRegistered', () => {
  const spec: WizardBackupSpec = {
    classRef: { name: 'pxc' },
    enabled: true,
    storages: [{ name: 's3-a', storageRef: { name: 's3-a' }, schedules: [] }],
  };

  it('returns the spec unchanged when there is no storage to register', () => {
    expect(ensureStorageRegistered(spec, undefined, 'pxc')).toBe(spec);
    expect(
      ensureStorageRegistered(undefined, undefined, 'pxc')
    ).toBeUndefined();
  });

  it('leaves the spec untouched when the storage is already registered', () => {
    expect(ensureStorageRegistered(spec, 's3-a', 'pxc')).toBe(spec);
  });

  it('appends the seeding storage when missing', () => {
    const result = ensureStorageRegistered(spec, 's3-b', 'pxc');
    expect(result?.storages.map((s) => s.storageRef.name)).toEqual([
      's3-a',
      's3-b',
    ]);
  });

  it('builds a minimal spec when none exists', () => {
    expect(ensureStorageRegistered(undefined, 's3-b', 'pxc')).toEqual({
      classRef: { name: 'pxc' },
      enabled: true,
      storages: [{ name: 's3-b', storageRef: { name: 's3-b' }, schedules: [] }],
    });
  });

  it('cannot build a spec without a class', () => {
    expect(
      ensureStorageRegistered(undefined, 's3-b', undefined)
    ).toBeUndefined();
  });
});
