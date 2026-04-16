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

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
// import { useUpdateDbClusterWithConflictRetry } from '../db-cluster/useUpdateDbCluster';
import {
  getDbInstanceQueryKey,
  getDbInstancesQueryKey,
  useDeleteDbInstance,
} from '../db-instances';
import { GetDbInstancesPayload, Instance } from 'shared-types/api.types';

export const useDbInstanceActions = (dbInstance: Instance) => {
  const clusterName = 'main';
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [openDetailsDialog, setOpenDetailsDialog] = useState(false);
  const [openRestoreDialog, setOpenRestoreDialog] = useState(false);
  const deleteMutation = useDeleteDbInstance(dbInstance.metadata?.name || '');
  const { mutate: deleteDbInstance } = deleteMutation;
  // const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
  //   dbInstance,
  //   {
  //     onSuccess: (updatedObject) => {
  //       queryClient.setQueryData<GetDbClusterPayload | undefined>(
  //         [DB_CLUSTERS_QUERY_KEY, updatedObject.metadata.namespace],
  //         (oldData) => {
  //           if (!oldData) {
  //             return undefined;
  //           }

  //           return {
  //             ...oldData,
  //             items: oldData.items.map((value) =>
  //               value.metadata.name === updatedObject.metadata.name
  //                 ? updatedObject
  //                 : value
  //             ),
  //           };
  //         }
  //       );
  //       enqueueSnackbar(
  //         updatedObject.spec.paused
  //           ? Messages.responseMessages.pause
  //           : Messages.responseMessages.resume,
  //         {
  //           variant: 'success',
  //         }
  //       );
  //     },
  //   }
  // );
  // const { mutate: restartDbCluster } = useUpdateDbClusterWithConflictRetry(
  //   dbCluster,
  //   {
  //     onSuccess: (updatedObject) => {
  //       queryClient.setQueryData<GetDbClusterPayload | undefined>(
  //         [DB_CLUSTERS_QUERY_KEY, updatedObject.metadata.namespace],
  //         (oldData) => {
  //           if (!oldData) {
  //             return undefined;
  //           }

  //           return {
  //             ...oldData,
  //             items: oldData.items.map((value) =>
  //               value.metadata.name === updatedObject.metadata.name
  //                 ? updatedObject
  //                 : value
  //             ),
  //           };
  //         }
  //       );
  //       enqueueSnackbar(Messages.responseMessages.restart, {
  //         variant: 'success',
  //       });
  //     },
  //   }
  // );
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // const isPaused = (dbCluster: DbCluster) => dbCluster.spec.paused;

  // const handleDbSuspendOrResumed = (dbCluster: DbCluster) => {
  //   const shouldBePaused = !isPaused(dbCluster);
  //   updateCluster(setDbClusterPausedStatus(dbCluster, shouldBePaused));
  // };

  // const handleDbRestart = (dbCluster: DbCluster) => {
  //   restartDbCluster(setDbClusterRestart(dbCluster));
  // };

  const handleDeleteDbInstance = () => {
    setOpenDeleteDialog(true);
  };

  const handleCloseDeleteDialog = (redirect?: string) => {
    setOpenDeleteDialog(false);

    if (redirect) {
      navigate(redirect);
    }
  };

  const handleConfirmDelete = (
    // TODO 1942 check if needed for instance deletion API.
    _keepBackupStorageData: boolean,
    onSuccess?: () => void
  ) => {
    deleteDbInstance(
      {
        // TODO
        dbInstanceName: dbInstance.metadata?.name || '',
        namespace: dbInstance.metadata?.namespace || '',
        // TODO 1942 check if needed
        // cleanupBackupStorage: !keepBackupStorageData,
      },
      {
        onSuccess: (_, variables) => {
          queryClient.setQueryData<GetDbInstancesPayload | undefined>(
            getDbInstancesQueryKey(variables.namespace, clusterName),
            (oldData) => {
              if (!oldData) {
                return undefined;
              }

              return {
                ...oldData,
                items: oldData.items?.map((item) => {
                  if (item.metadata?.name === variables.dbInstanceName) {
                    return {
                      ...item,
                      status: {
                        ...item.status,
                        // TODO v2 check should be deleted or not
                        // conditions: item.status?.conditions || [],
                        // crVersion: item.status?.crVersion || '',
                        // hostname: item.status?.hostname || '',
                        // port: item.status?.port || 0,
                        phase: 'Terminating',
                      },
                    };
                  }

                  return item;
                }),
              };
            }
          );
          queryClient.setQueryData<Instance | undefined>(
            getDbInstanceQueryKey(
              variables.namespace,
              variables.dbInstanceName,
              clusterName
            ),
            (oldData) =>
              oldData
                ? {
                    ...oldData,
                    status: {
                      ...oldData.status,
                      phase: 'Terminating',
                      // TODO v2 check should be deleted or not
                      // conditions: oldData.status?.conditions || [],
                      // hostname: oldData.status?.hostname || '',
                      // port: oldData.status?.port || 0,
                      // crVersion: oldData.status?.crVersion || '',
                    },
                  }
                : undefined
          );
          if (onSuccess) {
            onSuccess();
          } else {
            handleCloseDeleteDialog();
          }
        },
      }
    );
  };

  const handleRestoreDbCluster = () => {
    setOpenRestoreDialog(true);
  };

  const handleOpenDbDetailsDialog = () => {
    setOpenDetailsDialog(true);
  };

  const handleCloseRestoreDialog = () => {
    setOpenRestoreDialog(false);
  };

  const handleCloseDetailsDialog = () => {
    setOpenDetailsDialog(false);
  };

  return {
    openDeleteDialog,
    openRestoreDialog,
    openDetailsDialog,
    // handleDbSuspendOrResumed,
    // handleDbRestart,
    handleDeleteDbInstance,
    handleConfirmDelete,
    handleOpenDbDetailsDialog,
    handleCloseDeleteDialog,
    handleCloseDetailsDialog,
    // isPaused,
    handleRestoreDbCluster,
    handleCloseRestoreDialog,
    setOpenDetailsDialog,
    deleteMutation,
  };
};
