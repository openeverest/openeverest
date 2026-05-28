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
// TODO: Re-enable PITR UI when PITR restore flow is implemented.
// import { Alert } from '@mui/material';
// import { DbType } from '@percona/types';
import { LoadableChildren, RadioGroup, SelectInput } from '@percona/ui-lib';
// TODO: Re-enable PITR UI when PITR restore flow is implemented.
// import { DateTimePickerInput } from '@percona/ui-lib';
// import { dbEngineToDbType } from '@percona/utils';
// import ActionableAlert from 'components/actionable-alert';
import { DATE_FORMAT } from 'consts';
// TODO: Re-enable PITR UI when PITR restore flow is implemented.
// import { PITR_DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import { useFormContext } from 'react-hook-form';
import { Messages } from './restore-db-modal.messages';
import { BackupTypeValues, RestoreDbFields } from './restore-db-modal-schema';
import { ModalContentProps } from './restore-db-modal.types';

export const ModalContent = ({
  isLoading,
  header,
  succeededBackups,
}: ModalContentProps) => {
  const { watch } = useFormContext();
  // TODO: Re-enable PITR UI when PITR restore flow is implemented.
  // const { watch, resetField, setValue, getValues } = useFormContext();
  const backupType: BackupTypeValues = watch(RestoreDbFields.backupType);
  // TODO: Re-enable PITR UI when PITR restore flow is implemented.
  // useEffect(() => {
  //   if (!pitrData) {
  //     return;
  //   }
  //   if (!getValues(RestoreDbFields.pitrBackup)) {
  //     setValue(RestoreDbFields.pitrBackup, pitrData.latestDate);
  //   }
  // }, [getValues, pitrData, setValue]);

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
            // TODO: Re-enable PITR UI when PITR restore flow is implemented.
            // radioProps: {
            //   onClick: () => {
            //     resetField(RestoreDbFields.pitrBackup, {
            //       keepError: false,
            //       defaultValue: getValues(RestoreDbFields.pitrBackup),
            //     });
            //   },
            // },
          },
          {
            label: Messages.fromPitr,
            value: BackupTypeValues.fromPitr,
            // TODO: Re-enable PITR UI when PITR restore flow is implemented.
            // radioProps: {
            //   onClick: () => {
            //     resetField(RestoreDbFields.backupName, {
            //       keepError: false,
            //     });
            //   },
            // },
            // TODO: Re-enable PITR availability logic from main when PITR is implemented.
            // disabled:
            //   !!backupName &&
            //   pitrData?.latestBackupName !== watch(RestoreDbFields.backupName),
            disabled: true,
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
          {/* TODO: Re-enable main backup list ordering/filtering when PITR is implemented.
          {backups
            .filter((value) => value.state === BackupStatus.OK)
            .sort((a, b) => {
              if (a.created && b.created) {
                return new Date(b.created).valueOf() - new Date(a.created).valueOf();
              }
              return -1;
            })
            .map((value) => {
              const valueWithTime = `${value.name} - ${format(value.created!, DATE_FORMAT)}`;
              return (
                <MenuItem key={value.name} value={value.name}>
                  {valueWithTime}
                </MenuItem>
              );
            })}
          */}
        </SelectInput>
      )}

      {/* TODO: Re-enable PITR UI block from main when PITR restore flow is implemented. */}
      {/*
      {backupType === BackupTypeValues.fromPitr && (
        <>
          {pitrData && dbType === DbType.Postresql && (
            <ActionableAlert
              sx={{ mt: 1.5 }}
              message={Messages.pitrLimitationAlert}
              buttonMessage={Messages.seeDocs}
              onClick={() =>
                window.open(
                  'https://openeverest.io/documentation/current/reference/known_limitations.html#postgresql-limitation-for-pitr',
                  '_blank',
                  'noopener'
                )
              }
              buttonProps={{
                sx: { whiteSpace: 'nowrap' },
              }}
            />
          )}
          {pitrData && (
          <Alert
            sx={{ mt: 1.5, mb: 1.5 }}
            severity={pitrData?.gaps ? 'error' : 'info'}
          >
            {pitrData?.gaps
              ? Messages.gapDisclaimer
              : Messages.pitrDisclaimer(
                  format(pitrData?.earliestDate || new Date(), PITR_DATE_FORMAT),
                  format(pitrData?.latestDate || new Date(), PITR_DATE_FORMAT),
                  backupStorageName || ''
                )}
          </Alert>
          )}

          {!pitrData?.gaps && (
            <DateTimePickerInput
              ampm={false}
              views={['year', 'month', 'day', 'hours', 'minutes', 'seconds']}
              timeSteps={{ hours: 1, minutes: 1, seconds: 1 }}
              disableFuture
              disabled={!pitrData}
              minDate={new Date(pitrData?.earliestDate || new Date())}
              maxDate={new Date(pitrData?.latestDate || new Date())}
              format={PITR_DATE_FORMAT}
              name={RestoreDbFields.pitrBackup}
              label={pitrData ? 'Select point in time' : 'No options'}
              sx={{ mt: 1.5 }}
            />
          )}
        </>
      )}
      */}
    </LoadableChildren>
  );
};
