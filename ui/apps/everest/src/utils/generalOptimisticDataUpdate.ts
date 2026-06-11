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
