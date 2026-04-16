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

import { useQuery } from '@tanstack/react-query';
import { getProvidersFn } from 'api/providers';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { GetProviders, Provider } from 'shared-types/api.types';

export const useProviders = (
  options?: PerconaQueryOptions<GetProviders, unknown, Provider[]>
) => {
  return useQuery<GetProviders, unknown, Provider[]>({
    queryKey: ['providers'],
    queryFn: () => getProvidersFn(),
    retry: 3,
    refetchInterval: 10 * 1000,
    ...options,
    select: (providers) => {
      const selectedProviders = options?.select
        ? options.select(providers)
        : providers.items;

      return (selectedProviders ?? []).filter(
        (provider): provider is Provider => Boolean(provider)
      );
    },
  });
};
