import { AxiosError } from 'axios';
import { useRef } from 'react';
import {
  MutateOptions,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { updateDbInstanceFn } from 'api/instanceApi';
import { Instance } from 'types/api';
import { DB_INSTANCE_QUERY_KEY, useDbInstance } from './useDbInstance';

const UPDATE_RETRY_TIMEOUT_MS = 5000;
const UPDATE_RETRY_DELAY_MS = 200;

// TODO change to global use of cluster name during implementing multicluster feature
const CLUSTER_NAME = 'main';

export const updateDbInstance = (
  namespace: string,
  instanceName: string,
  data: Instance
) => updateDbInstanceFn(CLUSTER_NAME, namespace, instanceName, data);
// TODO check cron converter during backups implementation

export const useUpdateDbInstanceWithConflictRetry = (
  oldInstanceData: Instance,
  mutationOptions?: MutateOptions<
    Instance,
    AxiosError<unknown, unknown>,
    Instance,
    unknown
  >
) => {
  const {
    onSuccess: ownOnSuccess = () => {},
    onError: ownOnError = () => {},
    ...restMutationOptions
  } = mutationOptions || {};

  // Instance metadata is typed as Record<string, never> in the generated types,
  // but at runtime it contains standard Kubernetes fields.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const meta = oldInstanceData.metadata as any;
  const instanceName: string = meta?.name ?? '';
  const namespace: string = meta?.namespace ?? '';

  const queryClient = useQueryClient();
  const watchStartTime = useRef<number | null>(null);
  const dataToBeSent = useRef<Instance | null>(null);

  const { refetch } = useDbInstance(namespace, instanceName, {
    enabled: false,
  });

  const mutationMethods = useMutation<
    Instance,
    AxiosError,
    Instance,
    unknown
  >({
    mutationFn: (instance: Instance) => {
      dataToBeSent.current = instance;
      return updateDbInstance(namespace, instanceName, instance);
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
            { variant: 'error' }
          );
          ownOnError?.(error, vars, ctx);
          watchStartTime.current = null;
          return;
        }

        return new Promise<void>((resolve) =>
          setTimeout(async () => {
            const { data: freshInstance } = await refetch();

            if (freshInstance) {
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              const freshMeta = freshInstance.metadata as any;
              const { resourceVersion } = freshMeta;

              resolve();
              mutationMethods.mutate({
                ...dataToBeSent.current!,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                metadata: { ...freshMeta, resourceVersion } as any,
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
      queryClient.setQueryData<Instance>(
        [DB_INSTANCE_QUERY_KEY, namespace, instanceName],
        () => data
      );
      ownOnSuccess?.(data, vars, ctx);
    },
    ...restMutationOptions,
  });

  return mutationMethods;
};
