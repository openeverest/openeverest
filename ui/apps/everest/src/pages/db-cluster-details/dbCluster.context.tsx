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

import React, { createContext, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { QueryObserverResult, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import {
  getDbInstancesQueryKey,
  getDbInstanceQueryKey,
  useDbInstance,
} from 'hooks/api/db-instances';
import { DbInstanceContextProps } from './dbCluster.context.types';
import { Instance } from 'shared-types/api.types';
// import { useRBACPermissions } from 'hooks/rbac';

export const DbInstanceContext = createContext<DbInstanceContextProps>({
  instance: {} as Instance,
  isLoading: false,
  instanceDeleted: false,
  //   canReadBackups: false,
  canReadCredentials: false,
  //   canUpdateDb: false,
  //   temporarilyIncreaseInterval: () => {},
  //   queryResult: {} as QueryObserverResult<DbCluster, unknown>,
});

export const DbInstanceContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = 'main';
  const defaultInterval = 5 * 1000;
  const [refetchInterval] = useState(defaultInterval);
  const [instanceDeleted, setInstanceDeleted] = useState(false);
  const isDeleting = useRef(false);
  const queryClient = useQueryClient();
  const queryResult: QueryObserverResult<Instance, unknown> = useDbInstance(
    namespace,
    instanceName,
    {
      enabled: !!namespace && !!instanceName && !instanceDeleted,
      refetchInterval: refetchInterval,
    }
  );

  const { data: instance, isLoading, error } = queryResult;

  // const temporarilyIncreaseInterval = (
  //   interval: number,
  //   timeoutTime: number
  // ) => {
  //   setRefetchInterval(interval);
  //   const a = setTimeout(() => {
  //     setRefetchInterval(defaultInterval), clearTimeout(a);
  //   }, timeoutTime);
  // };

  //  const { canRead: canReadBackups } = useRBACPermissions(
  //     'database-cluster-backups',
  //     `${namespace}/${dbClusterName}`
  //   );
  //TODO RBAC fix to instance
  // const { canRead: canReadCredentials } = useRBACPermissions(
  //   'database-cluster-credentials',
  //   `${namespace}/${instanceName}`
  // );
  const canReadCredentials = true;
  //   const { canUpdate: canUpdateDb } = useRBACPermissions(
  //     'database-clusters',
  //     `${dbCluster?.metadata.namespace}/${dbCluster?.metadata.name}`
  //   );

  useEffect(() => {
    if (instance?.status?.phase === 'Terminating') {
      isDeleting.current = true;
    }

    if (isDeleting.current && error) {
      const axiosError = error as AxiosError;
      const errorStatus = axiosError.response ? axiosError.response.status : 0;
      setInstanceDeleted(errorStatus === 404);
      queryClient.invalidateQueries({
        queryKey: getDbInstancesQueryKey(namespace, clusterName),
      });
      queryClient.refetchQueries({
        queryKey: getDbInstancesQueryKey(namespace, clusterName),
        type: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: getDbInstanceQueryKey(namespace, instanceName, clusterName),
      });
    }
  }, [instance?.status, error, namespace, instanceName, queryClient]);

  return (
    <DbInstanceContext.Provider
      value={{
        instance,
        isLoading,
        instanceDeleted,
        // canReadBackups,
        // canUpdateDb,
        canReadCredentials,
        // temporarilyIncreaseInterval,
        // queryResult,
      }}
    >
      {children}
    </DbInstanceContext.Provider>
  );
};
