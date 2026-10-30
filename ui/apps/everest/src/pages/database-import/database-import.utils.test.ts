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

import { AlertColor } from '@mui/material';
import { Provider } from 'shared-types/api.types';
import { Backup, BackupClass, BackupImport } from 'shared-types/backups.types';
import { ScanProgress } from './database-import.types';
import {
  findLatestBackupImport,
  getClassProviders,
  getImportBackupRows,
  getScanAlert,
} from './database-import.utils';

const externalBackup = (
  name: string,
  path: string,
  completedAt: string,
  deletionTimestamp?: string
): Backup => ({
  metadata: { name, deletionTimestamp },
  spec: {
    classRef: { name: 'psmdb-pbm' },
    storageRef: { name: 's3' },
    origin: {
      type: 'External',
      external: { path, startedAt: completedAt, completedAt },
    },
  },
});

const provider = (name: string): Provider => ({ metadata: { name }, spec: {} });

const backupClass = (supportedProviders?: string[]): BackupClass => ({
  metadata: { name: 'psmdb-pbm' },
  spec: { executionMode: 'ProviderManaged', supportedProviders },
});

describe('getImportBackupRows', () => {
  it('lists external backups newest first, skipping deleting ones', () => {
    const rows = getImportBackupRows([
      externalBackup('b-old', 'old', '2026-01-01T00:00:00Z'),
      externalBackup('b-new', 'new', '2026-03-01T00:00:00Z'),
      externalBackup('b-gone', 'gone', '2026-04-01T00:00:00Z', '2026-04-02'),
    ]);

    expect(rows.map((row) => row.name)).toEqual(['b-new', 'b-old']);
    expect(rows[0]).toEqual({
      name: 'b-new',
      path: 'new',
      completedAt: '2026-03-01T00:00:00Z',
    });
  });

  it('skips backups an instance produced', () => {
    const instanceBackup: Backup = {
      metadata: { name: 'b-instance' },
      spec: {
        classRef: { name: 'psmdb-pbm' },
        storageRef: { name: 's3' },
        origin: { type: 'Instance', instanceRef: { name: 'db' } },
      },
    };

    expect(getImportBackupRows([instanceBackup])).toEqual([]);
  });
});

describe('getClassProviders', () => {
  const providers = [provider('psmdb'), provider('pxc')];

  it('keeps only the providers the class supports', () => {
    expect(getClassProviders(backupClass(['psmdb']), providers)).toEqual([
      providers[0],
    ]);
  });

  it('treats a class without supported providers as supporting all', () => {
    expect(getClassProviders(backupClass(), providers)).toEqual(providers);
  });

  it('returns nothing without a class', () => {
    expect(getClassProviders(undefined, providers)).toEqual([]);
  });
});

describe('findLatestBackupImport', () => {
  const backupImport = (
    name: string,
    storage: string,
    backupClassName: string,
    creationTimestamp: string,
    deletionTimestamp?: string
  ): BackupImport => ({
    metadata: { name, creationTimestamp, deletionTimestamp },
    spec: {
      classRef: { name: backupClassName },
      storageRef: { name: storage },
    },
  });

  it('picks the newest scan of the storage with the class', () => {
    const latest = findLatestBackupImport(
      [
        backupImport('old', 's3', 'pbm', '2026-10-01T00:00:00Z'),
        backupImport('new', 's3', 'pbm', '2026-10-02T00:00:00Z'),
        backupImport('deleting', 's3', 'pbm', '2026-10-04T00:00:00Z', 'now'),
        backupImport('other-storage', 's3-b', 'pbm', '2026-10-03T00:00:00Z'),
        backupImport('other-class', 's3', 'job', '2026-10-03T00:00:00Z'),
      ],
      { storageName: 's3', backupClassName: 'pbm' }
    );

    expect(latest?.metadata?.name).toBe('new');
  });
});

describe('getScanAlert', () => {
  const idle = { isScanning: false, isStalled: false };
  const cases: [ScanProgress, AlertColor | undefined][] = [
    [idle, undefined],
    [{ ...idle, isScanning: true, state: 'Pending' }, undefined],
    [{ ...idle, isScanning: true, isStalled: true }, 'warning'],
    [{ ...idle, isScanning: true, state: 'Error', message: 'x' }, 'warning'],
    [{ ...idle, state: 'Failed', message: 'x' }, 'error'],
    [{ ...idle, state: 'Succeeded' }, 'success'],
  ];

  it.each(cases)('maps %o to %s', (progress, severity) => {
    expect(getScanAlert(progress)?.severity).toBe(severity);
  });
});
