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

import { useEffect } from 'react';
import { Alert, MenuItem } from '@mui/material';
import { DateTimePickerInput, RadioGroup, SelectInput } from '@percona/ui-lib';
import { format } from 'date-fns';
import { useFormContext, useWatch } from 'react-hook-form';
import { PITR_DATE_FORMAT } from 'consts';
import { Messages } from './restore-db-modal.messages';
import {
  RecoveryTargetValues,
  RestoreDbFields,
} from './restore-db-modal-schema';
import { RestorePitrStorageOption } from './restore-db-modal.types';
import { resolveActiveStorage } from './restore-pitr.utils';

interface RestorePitrFieldsProps {
  pitrStorages: RestorePitrStorageOption[];
}

export const RestorePitrFields = ({ pitrStorages }: RestorePitrFieldsProps) => {
  const { setValue } = useFormContext();
  const selectedStorage: string = useWatch({
    name: RestoreDbFields.pitrStorage,
  });
  const recoveryTarget: RecoveryTargetValues = useWatch({
    name: RestoreDbFields.recoveryTarget,
  });
  const pointInTimeDate: Date | null = useWatch({
    name: RestoreDbFields.pointInTimeDate,
  });

  const activeStorage = resolveActiveStorage(pitrStorages, selectedStorage);

  // Keep the form's storage selection in sync with the resolved active storage
  // so the submit payload always names a storage even when the selector is
  // hidden (single-storage case).
  useEffect(() => {
    if (activeStorage && selectedStorage !== activeStorage.name) {
      setValue(RestoreDbFields.pitrStorage, activeStorage.name);
    }
  }, [activeStorage, selectedStorage, setValue]);

  // Seed the picker with the latest restorable moment (rather than an empty,
  // invalid field) the first time the user opens the date target.
  useEffect(() => {
    const restorableWindow = activeStorage?.window;
    if (
      recoveryTarget === RecoveryTargetValues.date &&
      restorableWindow?.available &&
      restorableWindow.latest &&
      !pointInTimeDate
    ) {
      setValue(RestoreDbFields.pointInTimeDate, restorableWindow.latest);
    }
  }, [activeStorage, recoveryTarget, pointInTimeDate, setValue]);

  if (!activeStorage) {
    return null;
  }

  const pitrWindow = activeStorage.window;

  return (
    <>
      {pitrStorages.length > 1 && (
        <SelectInput
          label={Messages.selectStorage}
          name={RestoreDbFields.pitrStorage}
          selectFieldProps={{
            labelId: 'restore-pitr-storage',
            label: Messages.selectStorage,
          }}
        >
          {pitrStorages.map((storage) => (
            <MenuItem key={storage.name} value={storage.name}>
              {storage.name}
            </MenuItem>
          ))}
        </SelectInput>
      )}

      <Alert
        sx={{ mt: 1.5, mb: 1.5 }}
        severity={pitrWindow.available ? 'info' : 'error'}
      >
        {pitrWindow.available && pitrWindow.earliest && pitrWindow.latest
          ? Messages.pitrDisclaimer(
              format(pitrWindow.earliest, PITR_DATE_FORMAT),
              format(pitrWindow.latest, PITR_DATE_FORMAT)
            )
          : (pitrWindow.message ?? Messages.pitrUnavailable)}
      </Alert>

      {pitrWindow.available && (
        <>
          <RadioGroup
            name={RestoreDbFields.recoveryTarget}
            options={[
              {
                label: Messages.recoveryLatest,
                value: RecoveryTargetValues.latest,
              },
              {
                label: Messages.recoveryDate,
                value: RecoveryTargetValues.date,
              },
            ]}
          />
          {recoveryTarget === RecoveryTargetValues.date && (
            <DateTimePickerInput
              ampm={false}
              views={['year', 'month', 'day', 'hours', 'minutes', 'seconds']}
              timeSteps={{ hours: 1, minutes: 1, seconds: 1 }}
              minDateTime={pitrWindow.earliest}
              maxDateTime={pitrWindow.latest}
              format={PITR_DATE_FORMAT}
              name={RestoreDbFields.pointInTimeDate}
              label={Messages.selectPointInTime}
              sx={{ mt: 1.5 }}
            />
          )}
        </>
      )}
    </>
  );
};
