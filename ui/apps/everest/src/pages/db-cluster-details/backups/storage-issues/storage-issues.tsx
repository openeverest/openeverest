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

import { Alert, AlertTitle, Box } from '@mui/material';
import { CodeCopyBlock } from '@percona/ui-lib';
import { Instance } from 'shared-types/api.types';
import { Messages } from './storage-issues.messages';
import { describeStorageIssue } from './storage-issues.utils';

interface StorageIssuesProps {
  instance: Instance;
}

// The storages the instance can't back up to as configured, and why.
export const StorageIssues = ({ instance }: StorageIssuesProps) => (
  <>
    {(instance.status?.backup?.storages ?? [])
      .filter((storage) => !!storage.reason)
      .map(({ name, reason = '', message }) => (
        <Alert
          key={name}
          severity="warning"
          sx={{ mb: 2, '& > .MuiAlert-message': { width: '100%' } }}
          data-testid={`backup-storage-issue-${name}`}
        >
          <AlertTitle>{Messages.title(name)}</AlertTitle>
          {describeStorageIssue(reason)} {Messages.keepsLastSetup}
          <Box sx={{ mt: 1, fontWeight: 600 }}>{Messages.reasonLabel}</Box>
          <CodeCopyBlock message={message || reason} severity="warning" />
        </Alert>
      ))}
  </>
);
