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

import { render, screen } from '@testing-library/react';
import { Instance } from 'shared-types/api.types';
import { buildBackupInstance } from 'pages/db-cluster-details/backups/__mocks__/backups-mocks';
import { StorageIssues } from './storage-issues';
import { Messages } from './storage-issues.messages';

type StorageStatus = NonNullable<
  NonNullable<NonNullable<Instance['status']>['backup']>['storages']
>[number];

const instanceReporting = (storages: StorageStatus[]): Instance => ({
  ...buildBackupInstance([]),
  status: { backup: { storages } },
});

describe('StorageIssues', () => {
  it("warns about each storage the instance can't back up to, and why", () => {
    render(
      <StorageIssues
        instance={instanceReporting([
          { name: 's3-ok', location: 'backups/ns/db' },
          {
            name: 's3-wrong-class',
            reason: 'BackupStorageIncompatible',
            message: 'backup class psmdb-pbm cannot write to backup storage',
          },
          {
            name: 's3-new-reason',
            reason: 'QuotaExceeded',
            message: 'bucket quota exceeded',
          },
        ])}
      />
    );

    expect(
      screen.queryByTestId('backup-storage-issue-s3-ok')
    ).not.toBeInTheDocument();
    expect(
      screen.getByTestId('backup-storage-issue-s3-wrong-class')
    ).toHaveTextContent(Messages.reasons.BackupStorageIncompatible);
    expect(
      screen.getByTestId('backup-storage-issue-s3-new-reason')
    ).toHaveTextContent('bucket quota exceeded');
  });
});
