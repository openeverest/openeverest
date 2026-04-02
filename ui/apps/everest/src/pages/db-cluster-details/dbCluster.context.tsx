import React, { createContext, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { QueryObserverResult, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import {
  DB_INSTANCES_QUERY_KEY,
  useDbInstance,
  DB_INSTANCE_QUERY_KEY,
} from 'hooks/api/db-instances';
import { DbInstanceContextProps } from './dbCluster.context.types';
import { Instance } from 'types/api';
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
        queryKey: [DB_INSTANCES_QUERY_KEY, namespace],
      });
      queryClient.invalidateQueries({
        queryKey: [DB_INSTANCE_QUERY_KEY, namespace, instanceName],
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
