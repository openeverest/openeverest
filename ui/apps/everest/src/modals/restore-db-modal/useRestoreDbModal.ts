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

import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useDbInstance } from 'hooks/api/db-instances';
import {
  getRestoresListQueryKey,
  useCreateInstanceRestore,
} from 'hooks/api/restores/useInstanceRestores';
import { useClusterName } from 'hooks/api/useClusterName';
import { BackupStatus } from 'shared-types/backups.types';
import {
  defaultValues,
  RestoreDbFields,
  RestoreDbFormData,
  schema,
} from './restore-db-modal-schema';
import { Messages } from './restore-db-modal.messages';
import {
  RestorableBackupOption,
  RestorePitrStorageOption,
} from './restore-db-modal.types';
import {
  getBackupName,
  getMetadataCreationTimestamp,
  getSafeTimeValue,
} from './restore-db-modal.utils';
import { resolvePitrWindow } from 'utils/pitr';
import { resolveRestoreAction } from './resolve-restore-action';
import { buildRestoreNavigationRouteState } from 'pages/database-form/restore-navigation-state';

interface UseRestoreDbModalParams {
  instanceName: string;
  namespace: string;
  isNewClusterMode: boolean;
  preselectedBackupName?: string;
  closeModal: () => void;
}

export const useRestoreDbModal = ({
  instanceName,
  namespace,
  isNewClusterMode,
  preselectedBackupName,
  closeModal,
}: UseRestoreDbModalParams) => {
  const clusterName = useClusterName();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data: backups = [], isLoading } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    { enabled: !!instanceName }
  );

  const { data: instance } = useDbInstance(namespace, instanceName, {
    enabled: !!instanceName,
  });

  // PITR is offered — for both in-place restore and clone-to-new-DB — only once
  // the source instance's provider has reported a restorable window in status;
  // otherwise the option stays disabled.
  const pitrStorages = useMemo<RestorePitrStorageOption[]>(
    () =>
      (instance?.status?.backup?.storages ?? [])
        .filter((storage) => storage.pitr)
        .map((storage) => ({
          name: storage.name,
          window: resolvePitrWindow(storage.pitr),
        })),
    [instance]
  );

  const succeededBackups = useMemo<RestorableBackupOption[]>(
    () =>
      backups
        .filter((backup) => backup.status?.state === BackupStatus.SUCCEEDED)
        .map((backup) => ({
          name: getBackupName(backup),
          startedAt:
            backup.status?.startedAt ?? getMetadataCreationTimestamp(backup),
          storageName: backup.spec?.storageRef?.name,
        }))
        .filter((backup) => !!backup.name)
        .sort(
          (a, b) =>
            getSafeTimeValue(b.startedAt) - getSafeTimeValue(a.startedAt)
        ),
    [backups]
  );

  const restoreSchema = useMemo(() => schema(pitrStorages), [pitrStorages]);

  // Preselect the backup the restore was launched from, mirroring V1.
  const formDefaultValues = useMemo(
    () => ({
      ...defaultValues,
      [RestoreDbFields.backupName]: preselectedBackupName ?? '',
    }),
    [preselectedBackupName]
  );

  const { mutate: createRestore, isPending: submitting } =
    useCreateInstanceRestore(clusterName, namespace);

  const onSubmit = (data: RestoreDbFormData) => {
    const action = resolveRestoreAction(data, {
      instanceName,
      isNewClusterMode,
      pitrStorages,
      succeededBackups,
    });

    if (action.kind === 'navigate-new') {
      closeModal();
      navigate('/databases/new', {
        state: buildRestoreNavigationRouteState(
          instanceName,
          namespace,
          action.dataSource
        ),
      });
      return;
    }

    if (action.kind === 'restore') {
      createRestore(action.variables, {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: getRestoresListQueryKey(
              clusterName,
              namespace,
              instanceName
            ),
          });
          enqueueSnackbar(Messages.restoreStarted, { variant: 'success' });
          closeModal();
        },
      });
    }
  };

  return {
    isLoading,
    succeededBackups,
    pitrStorages,
    submitting,
    restoreSchema,
    formDefaultValues,
    onSubmit,
  };
};
