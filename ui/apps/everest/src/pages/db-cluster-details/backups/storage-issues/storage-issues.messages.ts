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

export const Messages = {
  title: (storage: string) => `Backups to ${storage} don't run as configured`,
  keepsLastSetup:
    'Until this is fixed, the instance keeps its last working setup for this storage, if it had one.',
  reasonLabel: 'Reason:',
  // What is wrong, by the reason the instance reports.
  reasons: {
    BackupStorageNotFound: "The storage doesn't exist in this namespace.",
    BackupStorageNotAccepted:
      "The storage hasn't been checked yet, or isn't valid. Its status in Settings shows why.",
    BackupStorageIncompatible:
      "The instance's backup class can't write to this storage.",
    BackupStorageLocationChanged:
      "The storage now points to another location than the one this instance's backups are in.",
    UnsupportedParameter:
      "The database engine doesn't support one of the storage's settings.",
    ImplementationUnsupported:
      "The database engine doesn't support the storage's implementation.",
    TLSOptionUnsupported:
      "The database engine doesn't support the storage's TLS settings.",
  } satisfies Record<string, string>,
};
