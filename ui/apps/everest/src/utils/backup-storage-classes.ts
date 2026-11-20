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

import { isSection, Section } from 'components/ui-generator/ui-generator.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import { BackupStorageClass } from 'shared-types/backupStorages.types';
import { ClassContract, toSectionMap } from 'utils/class-contract';

// The label the Secrets API ties a storage's credentials Secret to its class
// by.
export const BACKUP_STORAGE_CLASS_SECRET_LABEL =
  'openeverest.io/backup-storage-class';

// A BackupStorage can't use it, whatever its class lists.
const ANONYMOUS_AUTH_MODE = 'Anonymous';

// The class uiSchema groups its forms by what they fill: the storage's
// location, its parameters and the Secret of each auth mode.
export interface BackupStorageClassUiSchema {
  location?: Section;
  storage?: Section;
  credentials: Record<string, Section>;
}

export const getBackupStorageClassName = (storageClass: BackupStorageClass) =>
  storageClass.metadata?.name ?? '';

// The generated CRD types keep uiSchema opaque, so each form is narrowed here.
export const getBackupStorageClassUiSchema = (
  storageClass: BackupStorageClass
): BackupStorageClassUiSchema => {
  const uiSchema: unknown = storageClass.spec.uiSchema;
  if (!isPlainObject(uiSchema)) {
    return { credentials: {} };
  }
  return {
    location: isSection(uiSchema.location) ? uiSchema.location : undefined,
    storage: isSection(uiSchema.storage) ? uiSchema.storage : undefined,
    credentials: toSectionMap(uiSchema.credentials),
  };
};

export const toBackupStorageClassContract = (
  storageClass: BackupStorageClass
): ClassContract => {
  const name = getBackupStorageClassName(storageClass);
  return {
    name,
    title: storageClass.spec.displayName || name,
    description: storageClass.spec.description,
    authModes: storageClass.spec.authModes.filter(
      (mode) => mode.name !== ANONYMOUS_AUTH_MODE
    ),
    credentialsForms: getBackupStorageClassUiSchema(storageClass).credentials,
    credentialsSecretLabel: BACKUP_STORAGE_CLASS_SECRET_LABEL,
    credentialsSecretFilter: { backupStorageClass: name },
  };
};
