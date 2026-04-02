// everest
// Copyright (C) 2023 Percona LLC
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
import { SubmitHandler } from 'react-hook-form';

import {
  pitrEditDialogDefaultValues,
  pitrEditDialogPropsSchema,
  PitrEditFields,
  PitrEditModalFormType,
  PitrEditModalProps,
} from './edit-pitr.types';
import { SwitchInput } from '@percona/ui-lib';
import { dbEngineToDbType } from '@percona/utils';
import { DbType } from '@percona/types';
import PitrStorage from './pitr-storage';
import { Messages as PITRMessages } from 'pages/common/pitr.messages';
import { Messages } from './edit-pitr.messages';
import { Typography } from '@mui/material';

export const PitrEditModal = ({
  open,
  handleCloseModal,
  handleSubmitModal,
  dbCluster,
}: PitrEditModalProps) => {
  const dbType = dbEngineToDbType(dbCluster.spec.engine.type);
  const schedules = dbCluster.spec.backup?.schedules || [];
  const backup = dbCluster?.spec?.backup;
  const backupsEnabled = schedules.length > 0;
  const pitrEnabled =
    dbType === DbType.Postresql
      ? backupsEnabled
      : backup?.pitr?.enabled || false;

  const backupStorageName = dbCluster.spec.backup?.pitr?.backupStorageName;

  const onSubmit: SubmitHandler<PitrEditModalFormType> = ({
    enabled,
    storageLocation,
  }) => {
    handleSubmitModal(
      enabled,
      (typeof storageLocation === 'string'
        ? storageLocation
        : storageLocation!.name) || schedules[0].backupStorageName
    );
  };
  return (
    <FormDialog
      size="XL"
      dataTestId="edit-pitr"
      isOpen={open}
      closeModal={handleCloseModal}
      schema={pitrEditDialogPropsSchema()}
      headerMessage={Messages.modal.title}
      onSubmit={onSubmit}
      submitMessage={Messages.modal.save}
      defaultValues={pitrEditDialogDefaultValues(
        pitrEnabled,
        backupStorageName
      )}
    >
      {dbType === DbType.Mongo && (
        <Typography variant="body2">
          {Messages.firstStorageWillBeUsed}
        </Typography>
      )}
      <SwitchInput
        label={Messages.enablePITR}
        name={PitrEditFields.enabled}
        labelCaption={PITRMessages.pitrSwitchLabelCaption(
          dbType,
          schedules[0]?.backupStorageName || ''
        )}
        formControlLabelProps={{
          sx: {
            mt: 1,
          },
        }}
      />
      <PitrStorage dbCluster={dbCluster} />
    </FormDialog>
  );
};
