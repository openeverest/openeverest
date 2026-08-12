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

import { Instance } from 'shared-types/api.types';
import { InstanceBackupStorage } from '../backups.types';

export const BACKUP_CLASS_NAME = 'psmdb-managed';

// The generated Instance type is opaque; a single cast here keeps the tests
// free of scattered assertions.
export const buildBackupInstance = (
  storages: InstanceBackupStorage[]
): Instance =>
  ({
    spec: {
      backup: {
        enabled: true,
        classRef: { name: BACKUP_CLASS_NAME },
        storages,
      },
    },
  }) as unknown as Instance;

interface ProviderManagedConfig {
  supportsPITR: boolean;
  limits?: { maxPITREnabledStorages?: number };
}

export const buildProviderClass = (
  name: string = BACKUP_CLASS_NAME,
  providerManaged: ProviderManagedConfig = {
    supportsPITR: true,
    limits: { maxPITREnabledStorages: 1 },
  }
) => ({
  metadata: { name },
  spec: { providerManaged },
});
