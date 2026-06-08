import { QueryClient } from '@tanstack/react-query';

export type OptimisticQueryContext<T> = {
  queryKey: readonly unknown[];
  previousData?: T[];
};

export const snapshotQueryData = <T>(
  queryClient: QueryClient,
  queryKey: readonly unknown[]
): OptimisticQueryContext<T> => ({
  queryKey,
  previousData: queryClient.getQueryData<T[]>(queryKey),
});

export const rollbackQueryData = <T>(
  queryClient: QueryClient,
  context?: OptimisticQueryContext<T>
) => {
  if (context) {
    queryClient.setQueryData(context.queryKey, context.previousData);
  }
};

export const optimisticCreateBy = <T>(
  queryClient: QueryClient,
  queryKey: readonly unknown[],
  createdObject: T,
  isSame: (item: T) => boolean
) => {
  queryClient.setQueryData<T[]>(queryKey, (oldData = []) => [
    createdObject,
    ...oldData.filter((item) => !isSame(item)),
  ]);
};

export const optimisticEditBy = <T>(
  queryClient: QueryClient,
  queryKey: readonly unknown[],
  updatedObject: T,
  isSame: (item: T) => boolean
) => {
  queryClient.setQueryData<T[]>(queryKey, (oldData = []) =>
    oldData.map((item) => (isSame(item) ? updatedObject : item))
  );
};

export const optimisticDeleteBy = <T>(
  queryClient: QueryClient,
  queryKey: readonly unknown[],
  shouldDelete: (item: T) => boolean
) => {
  queryClient.setQueryData<T[]>(queryKey, (oldData = []) =>
    oldData.filter((item) => !shouldDelete(item))
  );
};

export const updateDataAfterEdit =
  (
    queryClient: QueryClient,
    queryKey: readonly unknown[],
    identifier: string | undefined = 'id'
  ) =>
  <T extends object>(updatedObject: T) => {
    queryClient.setQueryData(queryKey, (oldData?: T[]) => {
      return (oldData || []).map((value) =>
        // @ts-ignore
        value[identifier] === updatedObject[identifier] ? updatedObject : value
      );
    });
  };

export const updateDataAfterCreate =
  (queryClient: QueryClient, queryKey: readonly unknown[]) =>
  <T extends object>(createdObject: T) => {
    queryClient.setQueryData(queryKey, (oldData?: T[]) => {
      return [createdObject, ...(oldData || [])];
    });
  };

export const updateDataAfterDelete =
  (
    queryClient: QueryClient,
    queryKey: readonly unknown[],
    identifier: string | undefined = 'id'
  ) =>
  <T extends object>(_: T, objectId: string) => {
    queryClient.setQueryData(queryKey, (oldData?: T[]) => {
      // @ts-ignore
      return (oldData || []).filter((value) => value[identifier] !== objectId);
    });
  };
