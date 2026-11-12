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

import ActionableAlert from '..';
import { Messages } from './BackupsActionableAlert.messages';
import { BackupsActionableAlertProps } from './BackupsActionableAlert.types';
import { CreateEditModalStorage } from 'pages/settings/storage-locations/createEditModal/create-edit-modal';
import { useAddBackupStorage } from 'hooks/api/backup-storages/useAddBackupStorage';

const BackupsActionableAlert = ({
  namespace,
  message = Messages.noStoragesMessage,
}: BackupsActionableAlertProps) => {
  const { canAdd, isOpen, open, modalProps } = useAddBackupStorage(namespace);

  return (
    <>
      <ActionableAlert
        message={message}
        buttonMessage={Messages.addStorage}
        data-testid="no-storage-message"
        onClick={open}
        {...(!canAdd && { action: undefined })}
      />
      {isOpen && <CreateEditModalStorage {...modalProps} />}
    </>
  );
};

export default BackupsActionableAlert;
