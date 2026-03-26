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

import { useMutation, UseMutationOptions } from '@tanstack/react-query';
import { deleteDbInstanceFn } from 'api/instanceApi';

export type DeleteDbInstanceArgType = {
  dbInstanceName: string;
  namespace: string;
  // cleanupBackupStorage: boolean;
};
export const useDeleteDbInstance = (
  dbInstanceName: string,
  options?: UseMutationOptions<
    unknown,
    unknown,
    DeleteDbInstanceArgType,
    unknown
  >
) => {
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';

  return useMutation({
    mutationKey: ['deleteDbInstance', dbInstanceName],
    mutationFn: ({
      namespace,
      dbInstanceName /*cleanupBackupStorage*/,
    }: DeleteDbInstanceArgType) =>
      deleteDbInstanceFn(
        clusterName,
        dbInstanceName,
        namespace /*, cleanupBackupStorage*/
      ),
    ...options,
  });
};
