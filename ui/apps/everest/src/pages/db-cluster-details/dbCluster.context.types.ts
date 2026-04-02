import { Instance } from 'types/api';

export interface DbInstanceContextProps {
  instance?: Instance;
  isLoading: boolean;
  instanceDeleted: boolean;
  //   canReadBackups: boolean;
  //   canUpdateDb: boolean;
    canReadCredentials: boolean;
  //   queryResult: QueryObserverResult<DbCluster, unknown>;
  //   clusterDeleted: boolean;
  //   temporarilyIncreaseInterval: (interval: number, timeoutTime: number) => void;
}
