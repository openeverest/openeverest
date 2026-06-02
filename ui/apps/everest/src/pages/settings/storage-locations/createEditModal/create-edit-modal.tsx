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
import TlsAlert from 'components/tls-alert';
import { useMemo } from 'react';
import { SubmitHandler } from 'react-hook-form';
import { BackupStorageFormValues } from 'shared-types/backupStorages.types';
import { Messages } from '../storage-locations.messages';
import {
  StorageLocationsFields,
  storageLocationDefaultValues,
  storageLocationEditValues,
  storageLocationsSchema,
} from '../storage-locations.types';
import { CreateEditStorageForm } from './create-edit-form';
import { CreateEditModalStorageProps } from './create-edit-modal.types';

export const CreateEditModalStorage = ({
  open,
  handleCloseModal,
  handleSubmitModal,
  selectedStorageLocation,
  isLoading = false,
  prefillNamespace = '',
}: CreateEditModalStorageProps) => {
  const isEditMode = !!selectedStorageLocation;
  const schema = useMemo(
    () =>
      isEditMode
        ? storageLocationsSchema.partial({
            [StorageLocationsFields.accessKey]: true,
            [StorageLocationsFields.secretKey]: true,
          })
        : storageLocationsSchema,
    [isEditMode]
  );

  const defaultValues = useMemo(
    () =>
      selectedStorageLocation
        ? storageLocationEditValues(selectedStorageLocation)
        : storageLocationDefaultValues(prefillNamespace),
    [prefillNamespace, selectedStorageLocation]
  );

  const onSubmit: SubmitHandler<BackupStorageFormValues> = (data) => {
    handleSubmitModal(isEditMode, data);
  };

  return (
    <FormDialog
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
      {({ watch }) => (
        <>
          <CreateEditStorageForm isEditMode={isEditMode} />
          {!watch(StorageLocationsFields.verifyTLS) && (
            <TlsAlert sx={{ mt: 2 }} />
          )}
        </>
      )}
    </FormDialog>
  );
};
