import { DbCluster } from 'shared-types/dbCluster.types';
import { QueryObserverResult } from '@tanstack/react-query';

export interface DbClusterContextProps {
  dbCluster?: DbCluster;
  isLoading: boolean;
  canReadBackups: boolean;
  canCreateBackup: boolean;
  canUpdateDb: boolean;
  canReadCredentials: boolean;
  queryResult: QueryObserverResult<DbCluster, unknown>;
  clusterDeleted: boolean;
  temporarilyIncreaseInterval: (interval: number, timeoutTime: number) => void;
}
