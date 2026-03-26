import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { DbCluster, DbClusterStatus } from 'shared-types/dbCluster.types';
import { DB_CLUSTER_QUERY } from '../db-cluster/useDbCluster';
// import { useUpdateDbClusterWithConflictRetry } from '../db-cluster/useUpdateDbCluster';
import {
  mergeNewDbClusterData,
  // setDbClusterPausedStatus,
  // setDbClusterRestart,
} from 'utils/db';
import { DB_INSTANCES_QUERY_KEY, useDeleteDbInstance } from '../db-instances';
import { GetDbInstancesPayload, Instance } from 'types/api';

export const useDbInstanceActions = (dbInstance: Instance) => {
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

  const isPaused = (dbCluster: DbCluster) => dbCluster.spec.paused;

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
    redirect?: string
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
            [DB_INSTANCES_QUERY_KEY, variables.namespace],
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
                        status: DbClusterStatus.deleting,
                      },
                    };
                  }

                  return item;
                }),
              };
            }
          );
          queryClient.setQueryData<DbCluster>(
            [DB_CLUSTER_QUERY, dbInstance.metadata?.name],
            (oldData) => {
              if (!oldData) {
                return undefined;
              }

              return {
                ...mergeNewDbClusterData(undefined, oldData, false),
                status: {
                  ...oldData.status,
                  conditions: oldData.status?.conditions || [],
                  hostname: oldData.status?.hostname || '',
                  port: oldData.status?.port || 0,
                  crVersion: oldData.status?.crVersion || '',
                  status: DbClusterStatus.deleting,
                },
              };
            }
          );
          handleCloseDeleteDialog(redirect);
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
    isPaused,
    handleRestoreDbCluster,
    handleCloseRestoreDialog,
    setOpenDetailsDialog,
    deleteMutation,
  };
};
