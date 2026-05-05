// everest
// Copyright (C) 2023 Percona LLC
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

import { AxiosError } from 'axios';
import { useRef } from 'react';
import {
  MutateOptions,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { updateDbClusterFn } from 'api/dbClusterApi';
import { DbCluster } from 'shared-types/dbCluster.types';
import { DB_CLUSTER_QUERY, useDbCluster } from './useDbCluster';
import cronConverter from 'utils/cron-converter';
import { mergeNewDbClusterData } from 'utils/db';

const UPDATE_RETRY_TIMEOUT_MS = 5000;
const UPDATE_RETRY_DELAY_MS = 200;

export const updateDbCluster = (dbCluster: DbCluster) =>
  updateDbClusterFn(dbCluster.metadata.name, dbCluster.metadata.namespace, {
    ...dbCluster,
    spec: {
      ...dbCluster?.spec,
      ...(dbCluster?.spec?.backup?.schedules && {
        backup: {
          ...dbCluster?.spec?.backup,
          schedules: dbCluster?.spec?.backup?.schedules.map((schedule) => ({
            ...schedule,
            schedule: cronConverter(
              schedule.schedule,
              Intl.DateTimeFormat().resolvedOptions().timeZone,
              'UTC'
            ),
          })),
        },
      }),
    },
  });

export const useUpdateDbClusterWithConflictRetry = (
  oldDbClusterData: DbCluster,
  mutationOptions?: MutateOptions<
    DbCluster,
    AxiosError<unknown, unknown>,
    DbCluster,
    unknown
  >
) => {
  const {
    onSuccess: ownOnSuccess = () => {},
    onError: ownOnError = () => {},
    ...restMutationOptions
  } = mutationOptions || {};
  const { name: dbClusterName, namespace } = oldDbClusterData.metadata;

  const queryClient = useQueryClient();
  const watchStartTime = useRef<number | null>(null);
  const clusterDataToBeSent = useRef<DbCluster | null>(null);
  const { refetch } = useDbCluster(dbClusterName, namespace, {
    enabled: false,
  });

  const mutationMethods = useMutation<
    DbCluster,
    AxiosError,
    DbCluster,
    unknown
  >({
    mutationFn: (dbCluster: DbCluster) => {
      clusterDataToBeSent.current = dbCluster;
      return updateDbCluster(dbCluster);
    },
    onError: async (error, vars, ctx) => {
      const { status } = error;

      if (status === 409) {
        if (watchStartTime.current === null) {
          watchStartTime.current = Date.now();
        }

        const timeDiff = Date.now() - watchStartTime.current;

        if (timeDiff > UPDATE_RETRY_TIMEOUT_MS) {
          enqueueSnackbar(
            'There is a conflict with the current object definition.',
            {
              variant: 'error',
            }
          );
          ownOnError?.(error, vars, ctx);
          watchStartTime.current = null;
          return;
        }

        return new Promise<void>((resolve) =>
          setTimeout(async () => {
            const { data: freshDbCluster } = await refetch();

            if (freshDbCluster) {
              const { resourceVersion } = freshDbCluster.metadata;

              resolve();
              mutationMethods.mutate({
                ...clusterDataToBeSent.current!,
                metadata: { ...freshDbCluster.metadata, resourceVersion },
              });
            } else {
              watchStartTime.current = null;
              ownOnError?.(error, vars, ctx);
              resolve();
            }
          }, UPDATE_RETRY_DELAY_MS)
        );
      }

      ownOnError?.(error, vars, ctx);
      return;
    },
    onSuccess: (data, vars, ctx) => {
      watchStartTime.current = null;
      queryClient.setQueryData<DbCluster>(
        [DB_CLUSTER_QUERY, dbClusterName],
        (oldData) => mergeNewDbClusterData(oldData, data, false)
      );
      ownOnSuccess?.(data, vars, ctx);
    },
    ...restMutationOptions,
  });

  return mutationMethods;
};
