import { SplitHorizonDNSConfig } from 'shared-types/splitHorizon.types';
import { api } from './api';

export const getAllSplitHorizonDNSConfigs = async (
  namespace: string
): Promise<Array<SplitHorizonDNSConfig & { namespace: string }>> => {
  const response = await api.get<{ items: SplitHorizonDNSConfig[] }>(
    `namespaces/${namespace}/engine-features/split-horizon-dns-configs`
  );
  return response.data.items.map((config) => ({
    ...config,
    namespace,
  }));
};

export const getSplitHorizonDNSConfig = async (
  namespace: string,
  name: string
) => {
  const response = await api.get<SplitHorizonDNSConfig>(
    `namespaces/${namespace}/engine-features/split-horizon-dns-configs/${name}`
  );
  return response.data;
};

export const createSplitHorizonDNSConfig = async (
  namespace: string,
  config: SplitHorizonDNSConfig
) => {
  const response = await api.post<SplitHorizonDNSConfig>(
    `namespaces/${namespace}/engine-features/split-horizon-dns-configs`,
    config
  );
  return response.data;
};

export const updateSplitHorizonDNSConfig = async (
  namespace: string,
  name: string,
  baseDomain: string,
  config: {
    certificate?: {
      'ca.crt'?: string;
      'ca.key'?: string;
    };
  }
) => {
  const response = await api.patch<SplitHorizonDNSConfig>(
    `namespaces/${namespace}/engine-features/split-horizon-dns-configs/${name}`,
    {
      baseDomainNameSuffix: baseDomain,
      ...config,
    }
  );
  return response.data;
};

export const deleteSplitHorizonDNSConfig = async (
  namespace: string,
  name: string
) => {
  const response = await api.delete<SplitHorizonDNSConfig>(
    `namespaces/${namespace}/engine-features/split-horizon-dns-configs/${name}`
  );
  return response.data;
};
