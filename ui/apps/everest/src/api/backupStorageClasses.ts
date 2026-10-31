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
  BackupStorageClass,
  BackupStorageClassList,
} from 'shared-types/backupStorages.types';
import { api } from './api';

export const getBackupStorageClassesFn = async (
  cluster: string
): Promise<BackupStorageClass[]> => {
  const response = await api.get<BackupStorageClassList>(
    `clusters/${cluster}/backup-storage-classes`
  );
  return response.data?.items ?? [];
};

export const getBackupStorageClassFn = async (
  cluster: string,
  name: string
): Promise<BackupStorageClass> => {
  const response = await api.get<BackupStorageClass>(
    `clusters/${cluster}/backup-storage-classes/${name}`
  );
  return response.data;
};
