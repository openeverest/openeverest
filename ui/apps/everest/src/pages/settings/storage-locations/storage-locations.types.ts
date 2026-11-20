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
  BackupStorageStatus,
} from 'shared-types/backupStorages.types';

export interface BackupStorageRow {
  name: string;
  namespace: string;
  // The storage class's display name.
  type: string;
  // Bucket and prefix.
  location: string;
  status: BackupStorageStatus;
  raw: BackupStorageCRD;
}
