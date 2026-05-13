import {
  CreateRestorePayload,
  GetRestorePayload,
} from 'shared-types/restores.types';
import { api } from './api';

export const createDbClusterRestore = async (
  data: CreateRestorePayload,
  namespace: string
) => {
  const response = await api.post(
    `namespaces/${namespace}/database-cluster-restores`,
    data
  );

  return response.data;
};

export const getDbClusterRestores = async (
  namespace: string,
  dbClusterName: string
) => {
  const response = await api.get<GetRestorePayload>(
    `namespaces/${namespace}/database-clusters/${dbClusterName}/restores`
  );

  return response.data;
};

export const deleteRestore = async (namespace: string, restoreName: string) => {
  const response = await api.delete(
    `namespaces/${namespace}/database-cluster-restores/${restoreName}`
  );

  return response.data;
};
