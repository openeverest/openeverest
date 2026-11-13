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

import { useEffect, useRef, useState } from 'react';
import {
  get,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
} from 'react-hook-form';
import { FormHelperText, IconButton, Stack } from '@mui/material';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import ActionableLabeledContent from '../../../actionable-labeled-content';
import TextInput from '../text';
import { PAIR_INPUT_SX } from './key-value-array.constants';
import { KeyValueArrayProps } from './key-value-array.types';
import { Messages } from './key-value-array.messages';

// Edits a string map as rows of key and value.
export const KeyValueArray = ({
  name,
  label,
  keyPlaceholder = Messages.key,
  valuePlaceholder = Messages.value,
  addButtonText = Messages.add,
  disabled,
  helperText,
}: KeyValueArrayProps) => {
  const { control, trigger } = useFormContext();
  const { errors } = useFormState({ control, name });
  const { fields, append, remove } = useFieldArray({ control, name });
  const [focusNewRow, setFocusNewRow] = useState(false);
  const rows: unknown = useWatch({ control, name });
  const keys = JSON.stringify(
    (Array.isArray(rows) ? rows : []).map((row: unknown) =>
      typeof row === 'object' && row !== null && 'key' in row
        ? row.key
        : undefined
    )
  );
  const validatedKeys = useRef(keys);

  // A key's error can depend on other rows (e.g. a duplicate), so a change to
  // any key re-validates them all.
  useEffect(() => {
    if (validatedKeys.current !== keys) {
      validatedKeys.current = keys;
      trigger(name);
    }
  }, [keys, name, trigger]);

  const arrayError = get(errors, name);
  const errorMessage: string | undefined =
    arrayError?.root?.message ?? arrayError?.message;

  return (
    <ActionableLabeledContent
      label={label}
      actionButtonProps={{
        dataTestId: `${name}-add`,
        buttonText: addButtonText,
        disabled,
        onClick: () => {
          append({ key: '', value: '' }, { shouldFocus: false });
          setFocusNewRow(true);
        },
      }}
    >
      {fields.map((field, index) => (
        <Stack key={field.id} direction="row" alignItems="center" gap={1}>
          <TextInput
            name={`${name}.${index}.key`}
            textFieldProps={{
              placeholder: keyPlaceholder,
              disabled,
              // Only a row just added mounts with focus.
              autoFocus: focusNewRow && index === fields.length - 1,
              inputProps: { 'aria-label': `${Messages.key} ${index + 1}` },
              sx: PAIR_INPUT_SX,
            }}
          />
          <TextInput
            name={`${name}.${index}.value`}
            textFieldProps={{
              placeholder: valuePlaceholder,
              disabled,
              inputProps: { 'aria-label': `${Messages.value} ${index + 1}` },
              sx: PAIR_INPUT_SX,
            }}
          />
          <IconButton
            aria-label={`${Messages.remove} ${index + 1}`}
            disabled={disabled}
            onClick={() => remove(index)}
            sx={{ mt: 1 }}
          >
            <DeleteOutlineOutlinedIcon />
          </IconButton>
        </Stack>
      ))}
      {(errorMessage || helperText) && (
        <FormHelperText error={!!errorMessage}>
          {errorMessage ?? helperText}
        </FormHelperText>
      )}
    </ActionableLabeledContent>
  );
};
