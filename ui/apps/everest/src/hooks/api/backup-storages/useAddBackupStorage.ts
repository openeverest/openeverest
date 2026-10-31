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
import { BackupStorageSecretConflictError } from 'api/backupStorage';
import { useClusterName } from 'hooks/api/useClusterName';
import { useRBACPermissions } from 'hooks/rbac';
import { CreateEditModalStorageProps } from 'pages/settings/storage-locations/createEditModal/create-edit-modal.types';
import {
  BACKUP_STORAGES_QUERY_KEY,
  useCreateBackupStorage,
} from './useBackupStorages';

// Adds a storage to a namespace outside Settings, through the same
// "Add storage location" dialog.
export const useAddBackupStorage = (
  namespace: string,
  onAdded?: (storageName: string) => void
) => {
  const [isOpen, setIsOpen] = useState(false);
  const [secretConflict, setSecretConflict] =
    useState<BackupStorageSecretConflictError>();
  const { mutate: createBackupStorage, isPending } = useCreateBackupStorage();
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const { canCreate } = useRBACPermissions('backup-storages', `${namespace}/*`);

  const close = () => {
    setIsOpen(false);
    setSecretConflict(undefined);
  };
  const modalProps: CreateEditModalStorageProps = {
    open: isOpen,
    handleCloseModal: close,
    handleSubmitModal: (_, data) => {
      createBackupStorage(data, {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: [BACKUP_STORAGES_QUERY_KEY, clusterName, data.namespace],
          });
          close();
          onAdded?.(data.name);
        },
        onError: (error) =>
          setSecretConflict(
            error instanceof BackupStorageSecretConflictError
              ? error
              : undefined
          ),
      });
    },
    isLoading: isPending,
    prefillNamespace: namespace,
    secretConflict,
  };

  return {
    canAdd: canCreate,
    isOpen,
    open: () => setIsOpen(true),
    modalProps,
  };
};
