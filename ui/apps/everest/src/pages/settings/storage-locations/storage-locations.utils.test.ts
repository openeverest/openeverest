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

import { UseQueryResult } from '@tanstack/react-query';
import {
  BackupStorageCondition,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import { s3Class, s3Storage } from './storage-locations.fixtures';
import { BackupStorageStatus } from './storage-locations.types';
import {
  convertBackupStoragesPayloadToTableFormat,
  getBackupStorageStatus,
} from './storage-locations.utils';

const condition = (
  type: string,
  status: BackupStorageCondition['status'],
  reason: string,
  message = ''
): BackupStorageCondition => ({
  type,
  status,
  reason,
  message,
  lastTransitionTime: '2026-10-31T00:00:00Z',
});

const withConditions = (...conditions: BackupStorageCondition[]) =>
  s3Storage({}, { conditions });

describe('getBackupStorageStatus', () => {
  it('shows an accepted storage as ready', () => {
    expect(
      getBackupStorageStatus(
        withConditions(condition('Accepted', 'True', 'Accepted'))
      )
    ).toEqual({ status: BackupStorageStatus.Accepted, statusLabel: 'Ready' });
  });

  it('shows the reason of a rejected storage, with its message', () => {
    expect(
      getBackupStorageStatus(
        withConditions(
          condition(
            'Accepted',
            'False',
            'CredentialsSecretNotFound',
            'Secret "creds" not found'
          )
        )
      )
    ).toEqual({
      status: BackupStorageStatus.Rejected,
      statusLabel: 'CredentialsSecretNotFound',
      statusMessage: 'Secret "creds" not found',
    });
  });

  it('shows a storage waiting for its referrers as deleting', () => {
    expect(
      getBackupStorageStatus(
        withConditions(
          condition('Accepted', 'True', 'Accepted'),
          condition(
            'Deleting',
            'True',
            'WaitingForReferrers',
            'Referenced by Instance team-a/db'
          )
        )
      )
    ).toEqual({
      status: BackupStorageStatus.Deleting,
      statusLabel: 'Deleting',
      statusMessage: 'Referenced by Instance team-a/db',
    });
  });

  it('shows a storage without conditions as pending', () => {
    expect(getBackupStorageStatus(s3Storage()).status).toBe(
      BackupStorageStatus.Pending
    );
  });
});

describe('convertBackupStoragesPayloadToTableFormat', () => {
  const result = (data: BackupStorageCRD[]) =>
    ({ isSuccess: true, data }) as UseQueryResult<BackupStorageCRD[], Error>;

  it('shows the class display name, falling back to its name', () => {
    const storages = [
      s3Storage(),
      s3Storage({
        backupStorageClassRef: { name: 'azure' },
        location: { bucket: 'container' },
      }),
    ];

    const rows = convertBackupStoragesPayloadToTableFormat(
      [result(storages)],
      [s3Class]
    );

    expect(
      rows.map(({ backupStorageClass, location, authMode }) => [
        backupStorageClass,
        location,
        authMode,
      ])
    ).toEqual([
      ['S3-compatible object storage', 'my-backups/team-a/prod', 'AccessKey'],
      ['azure', 'container', 'AccessKey'],
    ]);
  });

  it('skips queries that have not succeeded', () => {
    expect(
      convertBackupStoragesPayloadToTableFormat(
        [{ isSuccess: false } as UseQueryResult<BackupStorageCRD[], Error>],
        [s3Class]
      )
    ).toEqual([]);
  });
});
