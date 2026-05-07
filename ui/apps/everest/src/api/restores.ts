import { RestoreList } from 'shared-types/restores.types';
import { api } from './api';

export const getInstanceRestores = async (
  clusterName: string,
  namespace: string,
  instanceName: string
): Promise<RestoreList> => {
  const response = await api.get<RestoreList>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/restores`
  );

  return response.data;
};

export const deleteRestore = async (
  clusterName: string,
  namespace: string,
  instanceName: string,
  restoreName: string
) => {
  const response = await api.delete(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/restores/${restoreName}`
  );

  return response.data;
};
