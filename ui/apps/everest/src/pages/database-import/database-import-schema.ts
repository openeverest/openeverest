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

import { z } from 'zod';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Messages } from './database-import.messages';

export enum DatabaseImportFields {
  namespace = 'namespace',
  storage = 'storage',
  backupClassName = 'backupClassName',
  providerName = 'providerName',
  backupName = 'backupName',
}

export const databaseImportSchema = z.object({
  [DatabaseImportFields.namespace]: z
    .string()
    .min(1, Messages.validation.namespace),
  // The storage input holds the selected BackupStorage object.
  [DatabaseImportFields.storage]: z
    .custom<BackupStorageCRD>()
    .nullable()
    .refine(
      (storage) => !!storage?.metadata?.name,
      Messages.validation.storage
    ),
  [DatabaseImportFields.backupClassName]: z
    .string()
    .min(1, Messages.validation.backupClass),
  [DatabaseImportFields.providerName]: z
    .string()
    .min(1, Messages.validation.provider),
  [DatabaseImportFields.backupName]: z
    .string()
    .min(1, Messages.validation.backup),
});

export type DatabaseImportFormValues = z.infer<typeof databaseImportSchema>;
