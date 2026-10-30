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

import { useFieldArray, useFormContext } from 'react-hook-form';
import { IconButton, Stack } from '@mui/material';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import ActionableLabeledContent from '../../../actionable-labeled-content';
import TextInput from '../text';
import { KeyValueArrayProps } from './key-value-array.types';
import { Messages } from './key-value-array.messages';

// Edits a string map as rows of key and value.
export const KeyValueArray = ({
  name,
  label,
  keyPlaceholder = Messages.key,
  valuePlaceholder = Messages.value,
}: KeyValueArrayProps) => {
  const { control } = useFormContext();
  const { fields, append, remove } = useFieldArray({ control, name });

  return (
    <ActionableLabeledContent
      label={label}
      actionButtonProps={{
        dataTestId: `${name}-add`,
        buttonText: Messages.add,
        onClick: () => append({ key: '', value: '' }),
      }}
    >
      {fields.map((field, index) => (
        <Stack key={field.id} direction="row" alignItems="center" gap={1}>
          <TextInput
            name={`${name}.${index}.key`}
            textFieldProps={{
              placeholder: keyPlaceholder,
              inputProps: { 'aria-label': `${Messages.key} ${index + 1}` },
              sx: { flex: 1, mt: 1 },
            }}
          />
          <TextInput
            name={`${name}.${index}.value`}
            textFieldProps={{
              placeholder: valuePlaceholder,
              inputProps: { 'aria-label': `${Messages.value} ${index + 1}` },
              sx: { flex: 1, mt: 1 },
            }}
          />
          <IconButton
            aria-label={`${Messages.remove} ${index + 1}`}
            onClick={() => remove(index)}
            sx={{ mt: 1 }}
          >
            <DeleteOutlineOutlinedIcon />
          </IconButton>
        </Stack>
      ))}
    </ActionableLabeledContent>
  );
};
