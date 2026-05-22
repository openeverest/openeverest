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

import { useQueries, useQuery, UseQueryOptions } from '@tanstack/react-query';
import { GetNamespacesPayload } from 'shared-types/namespaces.types';
import { getNamespacesFn } from 'api/namespaces';
import { useClusterName } from '../useClusterName';
import { dbEnginesQuerySelect } from '../db-engines/useDbEngines';
import { getDbEnginesFn } from 'api/dbEngineApi';
import { DbEngine } from 'shared-types/dbEngines.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { DbEngineType } from '@percona/types';
import { useCallback, useMemo } from 'react';

export const NAMESPACES_QUERY_KEY = 'namespace';

export const useNamespaces = (
  options?: PerconaQueryOptions<GetNamespacesPayload, unknown, string[]>
) => {
  const clusterName = useClusterName();
  return useQuery<GetNamespacesPayload, unknown, string[]>({
    queryKey: [NAMESPACES_QUERY_KEY, clusterName],
    queryFn: () => getNamespacesFn(clusterName),
    select: (namespaces) => namespaces.sort((a, b) => a.localeCompare(b)),
    ...options,
  });
};

export const useDBEnginesForNamespaces = (
  retrieveUpgradingEngines = false,
  options?: PerconaQueryOptions<DbEngine[], unknown, DbEngine[]>
) => {
  const { data: namespaces = [], isFetching: fetchingNamespaces } =
    useNamespaces({
      refetchInterval: 5 * 1000,
    });

  const queries = namespaces.map<
    UseQueryOptions<DbEngine[], unknown, DbEngine[]>
  >((namespace) => ({
    queryKey: ['dbEngines-multi', namespace],
    // We don't use "select" here so that our cache saves data already formatted
    // Otherwise, every render from components cause "select" to be called, which means new values on every render
    queryFn: async () => {
      const data = await getDbEnginesFn(namespace);

      return dbEnginesQuerySelect(data, retrieveUpgradingEngines);
    },
    refetchInterval: 5 * 1000,
    ...options,
  }));

  const queryResults = useQueries({
    queries,
  });

  const refetchAll = useCallback(() => {
    queryResults.forEach((result) => result.refetch());
  }, [queryResults]);

  const results = queryResults.map((item, i) => ({
    namespace: namespaces[i],
    ...item,
  }));
  return { results, refetchAll, fetchingNamespaces };
};

export interface DbEngineForNamedpaceExpanded {
  dbEngine?: DbEngine;
  namespace: string;
}
export interface DbEnginesForDbTypeExpanded {
  type: DbEngineType;
  available: boolean;
  dbEngines: DbEngineForNamedpaceExpanded[];
}
export const useDBEnginesForDbEngineTypes = (
  dbEngineType?: DbEngineType,
  options?: PerconaQueryOptions<DbEngine[], unknown, DbEngine[]>
): [
  dbEnginesFoDbEngineTypes: DbEnginesForDbTypeExpanded[],
  dbEnginesFoDbEngineTypesFetching: boolean,
  refetch: () => void,
] => {
  const { results: dbEnginesForNamespaces, refetchAll } =
    useDBEnginesForNamespaces(false, options);
  const dbEnginesFetching = dbEnginesForNamespaces.some(
    (result) => result.isLoading
  );

  const dbEngineTypes = useMemo(
    () =>
      dbEngineType
        ? [dbEngineType]
        : (Object.keys(DbEngineType) as Array<keyof typeof DbEngineType>).map(
            (type) => DbEngineType[type]
          ),
    [dbEngineType]
  );

  const availableDbEngineTypes = useMemo(() => {
    if (!dbEnginesFetching) {
      return dbEngineTypes.map((type) => {
        const dbEnginesForDbType = dbEnginesForNamespaces.reduce(
          (prevVal, currVal, idx) => {
            const namespaceHasDbEngineForDbType = currVal.data?.find(
              (dbEngine) => dbEngine?.type === type
            );
            if (idx === 0 || prevVal?.length === 0) {
              if (namespaceHasDbEngineForDbType) {
                return [
                  {
                    dbEngine: namespaceHasDbEngineForDbType,
                    namespace: currVal.namespace,
                  },
                ];
              } else {
                return [];
              }
            } else {
              if (namespaceHasDbEngineForDbType) {
                return [
                  ...prevVal,
                  {
                    dbEngine: namespaceHasDbEngineForDbType,
                    namespace: currVal.namespace,
                  },
                ];
              } else {
                return [...prevVal];
              }
            }
          },
          <{ dbEngine?: DbEngine; namespace: string }[]>[]
        );
        return {
          type: type,
          dbEngines: dbEnginesForDbType,
          //available at least in one namespace
          available: dbEnginesForDbType?.length > 0 ? true : false,
        };
      });
    } else {
      return dbEngineTypes.map((type) => ({
        type: type,
        dbEngines: [],
        available: false,
      }));
    }
  }, [dbEnginesFetching, dbEngineTypes, dbEnginesForNamespaces]);

  return [availableDbEngineTypes, dbEnginesFetching, refetchAll];
};

export const useNamespace = (
  namespace: string,
  options?: PerconaQueryOptions<
    GetNamespacesPayload,
    unknown,
    string | undefined
  >
) => {
  const clusterName = useClusterName();
  return useQuery<GetNamespacesPayload, unknown, string | undefined>({
    queryKey: [NAMESPACES_QUERY_KEY, clusterName],
    queryFn: () => getNamespacesFn(clusterName),
    select: (data) => data.find((item) => item === namespace),
    ...options,
  });
};
