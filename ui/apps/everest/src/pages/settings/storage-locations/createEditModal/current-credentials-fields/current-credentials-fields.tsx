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

import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import { BackupStorageFormValues } from 'shared-types/backupStorages.types';
import { Messages } from '../../storage-locations.messages';
import { StorageLocationsFields } from '../../storage-locations.types';
import { ClassFields } from '../class-fields';
import { CredentialsField } from '../create-edit-modal.utils';

interface CurrentCredentialsFieldsProps {
  fields: CredentialsField[];
  // Why the keys can't be changed here, if they can't.
  readOnlyReason?: string;
}

// The keys of the credentials the storage uses, replaced in place: the API
// never returns a Secret's keys, so they come from the auth mode. Required
// keys are known to be set; optional keys may or may not be.
export const CurrentCredentialsFields = ({
  fields,
  readOnlyReason,
}: CurrentCredentialsFieldsProps) => {
  const { control, setValue } = useFormContext<BackupStorageFormValues>();
  const [replacedKeys, removedKeys] = useWatch({
    control,
    name: ['replacedCredentialKeys', 'removedCredentialKeys'],
  });
  const readOnly = !!readOnlyReason;

  const setKeys = (
    name:
      | StorageLocationsFields.replacedCredentialKeys
      | StorageLocationsFields.removedCredentialKeys,
    keys: string[]
  ) => setValue(name, keys, { shouldValidate: true });

  const clearValue = (path: string) =>
    setValue(`${StorageLocationsFields.credentials}.${path}`, '', {
      shouldValidate: true,
    });

  const keyRow = (
    field: CredentialsField,
    status: string,
    action: string,
    onClick: () => void
  ) => (
    <Stack
      key={field.key}
      direction="row"
      alignItems="center"
      gap={1}
      sx={{ mt: 2 }}
      data-testid={`credentials-key-${field.path}`}
    >
      <Typography variant="body2" sx={{ flex: 1 }}>
        {field.label}
      </Typography>
      <Typography variant="caption" color="text.secondary">
        {status}
      </Typography>
      <Button size="small" disabled={readOnly} onClick={onClick}>
        {action}
      </Button>
    </Stack>
  );

  return (
    <>
      {readOnlyReason ? (
        <Alert severity="info" sx={{ mt: 2 }}>
          {readOnlyReason}
        </Alert>
      ) : (
        <Typography variant="caption" sx={{ mt: 2 }}>
          {Messages.createEditModal.replaceAppliesEverywhere}
        </Typography>
      )}
      {fields.map((field) => {
        if (field.requiredByMode && !replacedKeys.includes(field.path)) {
          return keyRow(
            field,
            Messages.createEditModal.configured,
            Messages.createEditModal.replace,
            () =>
              setKeys(StorageLocationsFields.replacedCredentialKeys, [
                ...replacedKeys,
                field.path,
              ])
          );
        }
        if (removedKeys.includes(field.path)) {
          return keyRow(
            field,
            Messages.createEditModal.removedOnSave,
            Messages.createEditModal.undo,
            () =>
              setKeys(
                StorageLocationsFields.removedCredentialKeys,
                removedKeys.filter((key) => key !== field.path)
              )
          );
        }

        const stopEditing = field.requiredByMode
          ? () =>
              setKeys(
                StorageLocationsFields.replacedCredentialKeys,
                replacedKeys.filter((key) => key !== field.path)
              )
          : () =>
              setKeys(StorageLocationsFields.removedCredentialKeys, [
                ...removedKeys,
                field.path,
              ]);
        return (
          <Stack
            key={field.key}
            direction="row"
            alignItems="flex-start"
            gap={1}
          >
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <ClassFields
                fields={[
                  {
                    ...field,
                    required: field.requiredByMode,
                    helperText: field.requiredByMode
                      ? field.helperText
                      : Messages.createEditModal.optionalKeepsValue,
                  },
                ]}
                root={StorageLocationsFields.credentials}
                readOnly={readOnly}
              />
            </Box>
            <Button
              size="small"
              sx={{ mt: 3.5 }}
              disabled={readOnly}
              onClick={() => {
                clearValue(field.path);
                stopEditing();
              }}
            >
              {field.requiredByMode
                ? Messages.createEditModal.keepCurrentValue
                : Messages.createEditModal.remove}
            </Button>
          </Stack>
        );
      })}
    </>
  );
};
