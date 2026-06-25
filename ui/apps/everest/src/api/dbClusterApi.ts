// everest
// Copyright (C) 2023 Percona LLC
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
  DbCluster,
  GetDbClusterCredentialsPayload,
  GetDbClusterPayload,
} from 'shared-types/dbCluster.types';
import { api } from './api';
import { DBClusterComponentsList } from 'shared-types/components.types';
import { DataImportJobs } from 'shared-types/dataImporters.types';

export const createDbClusterFn = async (data: DbCluster, namespace: string) => {
  const response = await api.post<DbCluster>(
    `namespaces/${namespace}/database-clusters`,
    data
  );

  return response.data;
};

export const createDbClusterSecretFn = async (
  dbClusterName: string,
  namespace: string,
  data: Record<string, string>
) => {
  const payload = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'Secret',
    metadata: { name: `everest-secrets-${dbClusterName}` },
    stringData: { ...data },
  };
  const response = await api.post(
    `namespaces/${namespace}/database-clusters/${dbClusterName}/secret`,
    payload
  );
  return response.data;
};

export const updateDbClusterFn = async (
  dbClusterName: string,
  namespace: string,
  data: DbCluster
) => {
  const response = await api.put(
    `namespaces/${namespace}/database-clusters/${dbClusterName}`,
    data,
    {
      disableNotifications: (e) => e.status === 409,
    }
  );

  return response.data;
};

export const getDbClustersFn = async (namespace: string) => {
  const response = await api.get<GetDbClusterPayload>(
    `namespaces/${namespace}/database-clusters`
  );
  return response.data;
};

export const getDbClusterCredentialsFn = async (
  dbClusterName: string,
  namespace: string
) => {
  const response = await api.get<GetDbClusterCredentialsPayload>(
    `namespaces/${namespace}/database-clusters/${dbClusterName}/credentials`
  );

  return response.data;
};

export const getDbClusterFn = async (
  dbClusterName: string,
  namespace: string
) => {
  const response = await api.get<DbCluster>(
    `namespaces/${namespace}/database-clusters/${dbClusterName}`,
    {
      disableNotifications: true,
    }
  );
  return response.data;
};

export const deleteDbClusterFn = async (
  dbClusterName: string,
  namespace: string,
  deletionPolicy?: 'Cascade' | 'Orphan'
) => {
  const params = deletionPolicy ? `?deletionPolicy=${deletionPolicy}` : '';
  const response = await api.delete<DbCluster>(
    `namespaces/${namespace}/database-clusters/${dbClusterName}${params}`
  );
  return response.data;
};

export const getDBClusterComponentsListFn = async (
  namespace: string,
  dbClusterName: string
) => {
  const response = await api.get<DBClusterComponentsList>(
    `namespaces/${namespace}/database-clusters/${dbClusterName}/components`
  );
  return response.data;
};

export const getDbClusterImports = async (
  namespace: string,
  dbName: string
) => {
  const response = await api.get<DataImportJobs>(
    `namespaces/${namespace}/database-clusters/${dbName}/data-import-jobs`
  );
  return response.data;
};
