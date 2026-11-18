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
  BackupStorageCRD,
  BackupStorageState,
} from 'shared-types/backupStorages.types';
import {
  describeBackupStorageStatus,
  getBackupStorageStatus,
} from './backup-storages';
import { Messages } from './backup-storages.messages';

const condition = (
  type: string,
  status: 'True' | 'False' | 'Unknown',
  reason = '',
  message = ''
) => ({ type, status, reason, message, lastTransitionTime: '' });

const storage = (
  conditions: ReturnType<typeof condition>[],
  deletionTimestamp?: string
): BackupStorageCRD => ({
  metadata: { name: 's3-prod', deletionTimestamp },
  spec: {
    backupStorageClassRef: { name: 's3' },
    authentication: { mode: 'AccessKey' },
    location: { objectStore: { bucket: 'backups' } },
  },
  status: { conditions },
});

describe('getBackupStorageStatus', () => {
  it.each([
    ['accepted', [condition('Accepted', 'True', 'Accepted')], 'valid'],
    [
      'rejected',
      [condition('Accepted', 'False', 'InvalidLocation')],
      'invalid',
    ],
    ['not checked yet', [], 'pending'],
    ['being checked', [condition('Accepted', 'Unknown')], 'pending'],
  ])('reads a storage %s as %s', (_, conditions, state) => {
    expect(getBackupStorageStatus(storage(conditions)).state).toBe(state);
  });

  it('reads a storage waiting for its users to go as deleting, however valid', () => {
    const waiting = storage([
      condition('Accepted', 'True', 'Accepted'),
      condition('Deleting', 'True', 'WaitingForReferrers', 'used by orders-db'),
    ]);

    expect(getBackupStorageStatus(waiting)).toEqual({
      state: BackupStorageState.Deleting,
      reason: 'WaitingForReferrers',
      message: 'used by orders-db',
    });
  });

  it('reads a storage marked for deletion as deleting before core says why', () => {
    const marked = storage(
      [condition('Accepted', 'True', 'Accepted')],
      '2026-11-16T10:00:00Z'
    );

    expect(getBackupStorageStatus(marked).state).toBe(
      BackupStorageState.Deleting
    );
  });
});

describe('describeBackupStorageStatus', () => {
  it("says a valid storage's connection isn't tested", () => {
    expect(
      describeBackupStorageStatus({ state: BackupStorageState.Valid })
    ).toBe(Messages.notTested);
  });

  it("explains a known reason in the user's terms over core's message", () => {
    expect(
      describeBackupStorageStatus({
        state: BackupStorageState.Invalid,
        reason: 'CredentialsNotFound',
        message: 'secret "creds" not found',
      })
    ).toBe(Messages.reasons.CredentialsNotFound);
  });

  it("shows core's message for a reason this version doesn't know", () => {
    expect(
      describeBackupStorageStatus({
        state: BackupStorageState.Invalid,
        reason: 'QuotaExceeded',
        message: 'bucket quota exceeded',
      })
    ).toBe('bucket quota exceeded');
  });

  it("says a storage core hasn't looked at yet isn't checked", () => {
    expect(
      describeBackupStorageStatus({ state: BackupStorageState.Pending })
    ).toBe(Messages.notCheckedYet);
  });
});
