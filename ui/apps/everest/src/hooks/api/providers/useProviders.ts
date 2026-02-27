import { useQuery } from '@tanstack/react-query';
import { getProvidersFn } from 'api/providers';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { ProviderList } from 'types/api';

export const useProviders = (
  options?: PerconaQueryOptions<ProviderList, unknown, ProviderList>
) => {
  return useQuery<ProviderList, unknown, ProviderList>({
    queryKey: ['providers'],
    queryFn: () => getProvidersFn(),
    retry: 3,
    ...options,
  });
};
