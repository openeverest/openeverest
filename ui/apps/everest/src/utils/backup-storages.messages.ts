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

import { BackupStorageState } from 'shared-types/backupStorages.types';

export const BackupStorageStateMessages = {
  [BackupStorageState.Valid]: 'Valid',
  [BackupStorageState.Pending]: 'Checking',
  [BackupStorageState.Invalid]: 'Not valid',
  [BackupStorageState.Deleting]: 'Deleting',
} satisfies Record<BackupStorageState, string>;

export const Messages = {
  notCheckedYet: 'It has not been checked yet.',
  notTested:
    'Its settings are valid for its storage type. The connection to the storage is not tested.',
  // What is wrong, by the reason core gives.
  reasons: {
    ClassNotFound: 'Its storage type is not installed in this cluster.',
    InvalidClass:
      'Its storage type is misconfigured. Ask an administrator to fix it.',
    InvalidLocation: 'Its bucket or prefix breaks a rule of its storage type.',
    InvalidParameters: 'One of its settings breaks a rule of its storage type.',
    AuthModeUnsupported:
      "Its storage type doesn't support its authentication mode.",
    CredentialsNotFound:
      'Its credentials Secret, or a key it needs, is missing.',
    InvalidCABundle: "Its CA certificates are missing or aren't valid PEM.",
    WaitingForReferrers:
      'It is removed once no instance, backup, restore or import uses it.',
  } satisfies Record<string, string>,
};
