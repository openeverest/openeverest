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

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import semverCoerce from 'semver/functions/coerce';
import { DbCluster } from 'shared-types/dbCluster.types';
import { OperatorUpgradePendingAction } from 'shared-types/dbEngines.types';
import { changeDbClusterEngine, setDbClusterRestart } from 'utils/db';
import { updateDbCluster } from 'hooks/api/db-cluster/useUpdateDbCluster';
import { DB_CLUSTERS_QUERY_KEY } from 'hooks/api/db-clusters/useDbClusters';

const extractVersionFromMessage = (message: string): string | undefined => {
  for (const part of message.split(' ')) {
    if (semverCoerce(part, { includePrerelease: true })) return part;
  }
  return undefined;
};

export const useBatchResolvePendingActions = (
  namespace: string,
  dbClusters: DbCluster[],
  actionable: OperatorUpgradePendingAction[]
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => {
      const mutations = actionable
        .map((action) => {
          const cluster = dbClusters.find(
            (c) => c.metadata.name === action.name
          );
          if (!cluster) return null;

          if (action.pendingTask === 'restart') {
            return updateDbCluster(setDbClusterRestart(cluster));
          }

          if (action.pendingTask === 'upgradeEngine') {
            const version = extractVersionFromMessage(action.message);
            if (!version) return null;
            return updateDbCluster(changeDbClusterEngine(cluster, version));
          }

          return null;
        })
        .filter((p): p is Promise<DbCluster> => p !== null);

      return Promise.allSettled(mutations);
    },
    onSuccess: (results) => {
      const failed = results.filter((r) => r.status === 'rejected').length;
      const succeeded = results.length - failed;

      queryClient.invalidateQueries({
        queryKey: [DB_CLUSTERS_QUERY_KEY, namespace],
      });

      if (failed > 0) {
        enqueueSnackbar(
          `${succeeded} cluster(s) updated. ${failed} could not be updated — retry individually.`,
          { variant: 'warning' }
        );
      }
    },
    onError: () => {
      enqueueSnackbar('Failed to apply pending actions.', { variant: 'error' });
    },
  });
};
