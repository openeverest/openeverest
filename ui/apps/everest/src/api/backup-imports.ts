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

import { BackupImport, BackupImportList } from 'shared-types/backups.types';
import { api } from './api';

export const getBackupImportsFn = async (
  clusterName: string,
  namespace: string
): Promise<BackupImportList> => {
  const response = await api.get<BackupImportList>(
    `clusters/${clusterName}/namespaces/${namespace}/backup-imports`
  );

  return response.data;
};

export const createBackupImportFn = async (
  clusterName: string,
  namespace: string,
  backupImport: BackupImport
): Promise<BackupImport> => {
  const response = await api.post<BackupImport>(
    `clusters/${clusterName}/namespaces/${namespace}/backup-imports`,
    backupImport
  );

  return response.data;
};
