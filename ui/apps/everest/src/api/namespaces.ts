import { api } from './api';
import { GetNamespacesPayload } from 'shared-types/namespaces.types';

export const getNamespacesFn = async (clusterName: string) => {
  const response = await api.get<GetNamespacesPayload>(
    `clusters/${clusterName}/namespaces`
  );
  return response.data;
};
