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

import { FormDialog } from 'components/form-dialog/form-dialog';
import { useBackupStorageClasses } from 'hooks/api/backup-storage-classes';
import { useMemo } from 'react';
import { BackupStorageFormValues } from 'shared-types/backupStorages.types';
import { Messages } from '../storage-locations.messages';
import { CreateEditStorageForm } from './create-edit-form';
import { CreateEditModalStorageProps } from './create-edit-modal.types';
import {
  findBackupStorageClass,
  prepareStorageSubmitValues,
  storageLocationDefaultValues,
  storageLocationEditValues,
  storageLocationSchema,
} from './create-edit-modal.utils';

export const CreateEditModalStorage = ({
  open,
  handleCloseModal,
  handleSubmitModal,
  selectedStorageLocation,
  isLoading = false,
  prefillNamespace = '',
  secretConflict,
}: CreateEditModalStorageProps) => {
  const isEditMode = !!selectedStorageLocation;
  const { data, isLoading: loadingBackupStorageClasses } =
    useBackupStorageClasses();
  const backupStorageClasses = useMemo(() => data ?? [], [data]);

  const schema = useMemo(
    () => storageLocationSchema(backupStorageClasses, selectedStorageLocation),
    [backupStorageClasses, selectedStorageLocation]
  );

  const defaultValues = useMemo(
    () =>
      selectedStorageLocation
        ? storageLocationEditValues(selectedStorageLocation)
        : storageLocationDefaultValues(prefillNamespace),
    [prefillNamespace, selectedStorageLocation]
  );

  const onSubmit = (data: BackupStorageFormValues) =>
    handleSubmitModal(
      isEditMode,
      prepareStorageSubmitValues(
        data,
        findBackupStorageClass(backupStorageClasses, data.backupStorageClass),
        selectedStorageLocation
      )
    );

  return (
    <FormDialog<BackupStorageFormValues>
      size="XL"
      isOpen={open}
      closeModal={handleCloseModal}
      submitting={isLoading}
      headerMessage={Messages.createEditModal.addEditModal(isEditMode)}
      onSubmit={onSubmit}
      submitMessage={Messages.createEditModal.addEditButton(isEditMode)}
      schema={schema}
      defaultValues={defaultValues}
    >
      <CreateEditStorageForm
        isEditMode={isEditMode}
        backupStorageClasses={backupStorageClasses}
        loadingBackupStorageClasses={loadingBackupStorageClasses}
        previous={selectedStorageLocation}
        secretConflict={secretConflict}
      />
    </FormDialog>
  );
};
