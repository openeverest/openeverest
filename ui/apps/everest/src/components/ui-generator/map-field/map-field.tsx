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
import {
  get,
  useFieldArray,
  useFormContext,
  useFormState,
} from 'react-hook-form';
import { Box, FormHelperText, IconButton, Stack } from '@mui/material';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import { ActionableLabeledContent } from '@percona/ui-lib';
import { getMapRow } from 'components/ui-generator/utils/map-field';
import { useUiGeneratorContext } from '../ui-generator-context';
import { Messages } from './map-field.messages';
import { MapFieldProps } from './map-field.types';
import { toRowAtIndex } from './map-field.utils';
import { useRevalidateRowKeys } from './use-revalidate-row-keys';

// Edits a map as rows. Each row is the item template's fields, drawn by the UI
// Generator like any other field, so every param and rule of theirs applies.
export const MapField = ({
  item,
  name,
  children: renderRow,
}: MapFieldProps) => {
  const { fieldParams } = item;
  const { loadingDefaultsForEdition } = useUiGeneratorContext();
  const disabled = !!loadingDefaultsForEdition || !!fieldParams.disabled;
  const row = getMapRow(item);
  const { control } = useFormContext();
  const { errors } = useFormState({ control, name });
  const { fields, append, remove } = useFieldArray({ control, name });
  const [focusNewRow, setFocusNewRow] = useState(false);
  useRevalidateRowKeys(name);

  // autoFocus acts once, as the new row mounts. Dropping the flag right after
  // keeps a row that remounts later (indexes shift on a remove) from taking
  // the focus.
  useEffect(() => {
    if (focusNewRow) setFocusNewRow(false);
  }, [focusNewRow]);

  const mapError = get(errors, name);
  const errorMessage: string | undefined =
    mapError?.root?.message ?? mapError?.message;

  return (
    <ActionableLabeledContent
      label={fieldParams.label}
      actionButtonProps={{
        dataTestId: `${name}-add`,
        buttonText: fieldParams.addButtonLabel ?? Messages.add,
        disabled,
        onClick: () => {
          append({ key: '', value: '' }, { shouldFocus: false });
          setFocusNewRow(true);
        },
      }}
    >
      <Stack gap={1}>
        {fields.map((field, index) => (
          <Stack key={field.id} direction="row" alignItems="flex-start" gap={1}>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              {renderRow(
                toRowAtIndex(row, index, {
                  disabled,
                  focus: focusNewRow && index === fields.length - 1,
                }),
                `${name}.${index}`
              )}
            </Box>
            <IconButton
              aria-label={Messages.remove(index + 1)}
              disabled={disabled}
              onClick={() => remove(index)}
            >
              <DeleteOutlineOutlinedIcon />
            </IconButton>
          </Stack>
        ))}
      </Stack>
      {(errorMessage || fieldParams.helperText) && (
        <FormHelperText error={!!errorMessage}>
          {errorMessage ?? fieldParams.helperText}
        </FormHelperText>
      )}
    </ActionableLabeledContent>
  );
};
