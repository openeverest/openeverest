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

import { useEffect, useState } from 'react';
import { useBackupStorageClasses } from 'hooks/api/backup-storage-classes';
import { useBackupStorage } from 'hooks/api/backup-storages';
import { BackupStorageClass } from 'shared-types/backupStorages.types';
import { BackupStorageForm } from './backup-storage-form/backup-storage-form';
import { BackupStorageModalProps } from './backup-storage-modal.types';
import { useBackupStoragePermissions } from './hooks/use-backup-storage-permissions';

const NO_CLASSES: BackupStorageClass[] = [];

// Edits start from the storage as the server has it now, so the save carries
// its current resourceVersion.
export const BackupStorageModal = ({
  target,
  ...formProps
}: BackupStorageModalProps) => {
  const { onClose } = formProps;
  const [changedMeanwhile, setChangedMeanwhile] = useState(false);
  const { data: classes = NO_CLASSES, isLoading: classesLoading } =
    useBackupStorageClasses();
  const { createNamespaces, isLoading: permissionsLoading } =
    useBackupStoragePermissions();
  const {
    data: original,
    refetch,
    isError,
  } = useBackupStorage(target?.namespace ?? '', target?.name ?? '', {
    enabled: !!target,
    staleTime: 0,
    gcTime: 0,
  });

  // The request already reported the error; don't leave an invisible modal open.
  useEffect(() => {
    if (isError && !original) {
      onClose();
    }
  }, [isError, original, onClose]);

  // The classes are read once the user's permissions are known. A form shown
  // before that would be replaced as soon as they are.
  if (permissionsLoading || classesLoading || (target && !original)) {
    return null;
  }

  return (
    <BackupStorageForm
      {...formProps}
      original={original}
      classes={classes}
      namespaces={createNamespaces}
      changedMeanwhile={changedMeanwhile}
      onChangedMeanwhile={() => {
        setChangedMeanwhile(true);
        refetch();
      }}
    />
  );
};
