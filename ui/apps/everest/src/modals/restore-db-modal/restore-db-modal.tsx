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

import { FormDialog } from 'components/form-dialog';
import { Messages } from './restore-db-modal.messages';
import { ModalContent } from './modal-content';
import { RestoreDbModalProps } from './restore-db-modal.types';
import { useRestoreDbModal } from './useRestoreDbModal';

const RestoreDbModal = ({
  isOpen,
  closeModal,
  instanceName,
  namespace,
  isNewClusterMode = false,
  preselectedBackupName,
}: RestoreDbModalProps) => {
  const {
    isLoading,
    succeededBackups,
    pitrStorages,
    submitting,
    restoreSchema,
    formDefaultValues,
    onSubmit,
  } = useRestoreDbModal({
    instanceName,
    namespace,
    isNewClusterMode,
    preselectedBackupName,
    closeModal,
  });

  return (
    <FormDialog
      size="XXXL"
      isOpen={isOpen}
      dataTestId="restore-modal"
      closeModal={closeModal}
      headerMessage={
        isNewClusterMode ? Messages.headerMessageCreate : Messages.headerMessage
      }
      schema={restoreSchema}
      submitting={submitting}
      defaultValues={formDefaultValues}
      onSubmit={onSubmit}
      submitMessage={isNewClusterMode ? Messages.create : Messages.restore}
    >
      <ModalContent
        isLoading={isLoading}
        header={isNewClusterMode ? Messages.subHeadCreate : Messages.subHead}
        succeededBackups={succeededBackups}
        pitrStorages={pitrStorages}
      />
    </FormDialog>
  );
};

export default RestoreDbModal;
