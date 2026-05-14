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
  BackupList,
  CreateBackupPayload,
  CreateBackupResponse,
  DeleteBackupPayload,
  GetBackupClassPayload,
  GetBackupPayload,
  ListBackupClassesPayload,
  // DatabaseClusterPitrPayload,
} from 'shared-types/backups.types';
import { api } from './api';

export const getBackupFn = async (
  clusterName: string,
  namespace: string,
  backupName: string
) => {
  const response = await api.get<GetBackupPayload>(
    `clusters/${clusterName}/namespaces/${namespace}/backups/${backupName}`
  );

  return response.data;
};

export const createBackupOnDemandFn = async (
  clusterName: string,
  namespace: string,
  payload: CreateBackupPayload
) => {
  const response = await api.post<CreateBackupResponse>(
    `clusters/${clusterName}/namespaces/${namespace}/backups`,
    payload
  );

  return response.data;
};

export const deleteBackupFn = async (
  clusterName: string,
  namespace: string,
  backupName: string
) => {
  const response = await api.delete<DeleteBackupPayload>(
    `clusters/${clusterName}/namespaces/${namespace}/backups/${backupName}`
  );

  return response.data;
};

export const getBackupsListFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string
): Promise<BackupList> => {
  const response = await api.get<BackupList>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/backups`
  );

  return response.data;
};

export const getBackupClassesListFn = async (clusterName: string) => {
  const response = await api.get<ListBackupClassesPayload>(
    `clusters/${clusterName}/backup-classes`
  );

  return response.data;
};

export const getBackupClassFn = async (
  clusterName: string,
  backupClassName: string
) => {
  const response = await api.get<GetBackupClassPayload>(
    `clusters/${clusterName}/backup-classes/${backupClassName}`
  );

  return response.data;
};

// TODO: PITR is not part of this task — uncomment when working on PITR feature
// export const getPitrFn = async (dbClusterName: string, namespace: string) => {
//   const response = await api.get<DatabaseClusterPitrPayload>(
//     `/namespaces/${namespace}/database-clusters/${dbClusterName}/pitr`,
//     {
//       disableNotifications: true,
//     }
//   );
//
//   return response.data;
// };
