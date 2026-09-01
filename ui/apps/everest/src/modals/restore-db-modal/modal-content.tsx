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

import { MenuItem, Typography } from '@mui/material';
import { LoadableChildren, RadioGroup, SelectInput } from '@percona/ui-lib';
import { DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import { useWatch } from 'react-hook-form';
import { Messages } from './restore-db-modal.messages';
import { BackupTypeValues, RestoreDbFields } from './restore-db-modal-schema';
import { ModalContentProps } from './restore-db-modal.types';
import { RestorePitrFields } from './restore-pitr-fields';

export const ModalContent = ({
  isLoading,
  header,
  succeededBackups,
  pitrStorages,
}: ModalContentProps) => {
  const backupType: BackupTypeValues = useWatch({
    name: RestoreDbFields.backupType,
  });

  return (
    <LoadableChildren loading={isLoading}>
      <Typography variant="body1">{header}</Typography>
      <RadioGroup
        name={RestoreDbFields.backupType}
        radioGroupFieldProps={{
          sx: {
            ml: 1,
            display: 'flex',
            gap: 3,
            '& label': {
              display: 'flex',
              gap: '10px',
              alignItems: 'center',
              padding: 1,
              '& span': {
                padding: '0px !important',
              },
            },
          },
        }}
        options={[
          {
            label: Messages.fromBackup,
            value: BackupTypeValues.fromBackup,
          },
          {
            label: Messages.fromPitr,
            value: BackupTypeValues.fromPitr,
            disabled: pitrStorages.length === 0,
          },
        ]}
      />

      {backupType === BackupTypeValues.fromBackup && (
        <SelectInput
          label={Messages.selectBackup}
          name={RestoreDbFields.backupName}
          selectFieldProps={{
            labelId: 'restore-backup',
            label: Messages.selectBackup,
          }}
        >
          {succeededBackups.map((backup) => {
            const label = backup.startedAt
              ? `${backup.name} - ${format(new Date(backup.startedAt), DATE_FORMAT)}`
              : backup.name;
            return (
              <MenuItem key={backup.name} value={backup.name}>
                {label}
              </MenuItem>
            );
          })}
        </SelectInput>
      )}

      {backupType === BackupTypeValues.fromPitr && (
        <RestorePitrFields pitrStorages={pitrStorages} />
      )}
    </LoadableChildren>
  );
};
