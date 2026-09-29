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

import { useState } from 'react';
import { SxProps, TextField, Theme } from '@mui/material';
import { useController, useFormContext } from 'react-hook-form';
import { AffinityFormFields } from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/affinity-form.types';
import { Messages } from './group-editor-dialog.messages';

interface ConditionValuesInputProps {
  namePrefix?: string;
  disabled?: boolean;
  sx?: SxProps<Theme>;
}

const toValues = (text: string): string[] =>
  text
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean);

// k8s match values are a free list of strings. A comma-separated text field
// stays as compact as the sibling key/operator inputs (chips grow the row) and
// makes "how to finish a value" obvious. The raw text is kept locally so commas
// and spaces aren't reformatted mid-typing; the form model stays a string[].
// Errors are surfaced by the enclosing SegmentedField, not here.
export const ConditionValuesInput = ({
  namePrefix = '',
  disabled,
  sx,
}: ConditionValuesInputProps) => {
  const { control } = useFormContext();
  const { field } = useController({
    name: `${namePrefix}${AffinityFormFields.values}`,
    control,
  });
  const [text, setText] = useState(() =>
    Array.isArray(field.value) ? field.value.join(', ') : ''
  );

  return (
    <TextField
      label="Values"
      size="small"
      disabled={disabled}
      fullWidth
      placeholder={Messages.valuesPlaceholder}
      // SegmentedField hides the label (sr-only); force shrink so the
      // placeholder shows without focus, matching the sibling inputs.
      slotProps={{ inputLabel: { shrink: true } }}
      value={text}
      onChange={(event) => {
        setText(event.target.value);
        field.onChange(toValues(event.target.value));
      }}
      onBlur={field.onBlur}
      sx={sx}
    />
  );
};
