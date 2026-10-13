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

import { render, screen } from '@testing-library/react';
import { MenuItem, TextField } from '@mui/material';
import SegmentedField from './segmented-field';

describe('SegmentedField', () => {
  it('merges each control into one group, without dividers by default', () => {
    render(
      <SegmentedField label="Condition">
        <TextField label="Key" />
        <TextField select label="Operator" defaultValue="in">
          <MenuItem value="in">in</MenuItem>
        </TextField>
        <TextField label="Values" />
      </SegmentedField>
    );

    expect(
      screen.getByRole('group', { name: 'Condition' })
    ).toBeInTheDocument();
    // No dividers unless opted in.
    expect(screen.queryAllByRole('separator')).toHaveLength(0);
    // The child labels stay associated (a11y) even though hidden visually.
    expect(screen.getByLabelText('Key')).toBeInTheDocument();
    expect(screen.getByLabelText('Values')).toBeInTheDocument();
  });

  it('draws dividers between segments when asked', () => {
    render(
      <SegmentedField dividers>
        <TextField label="Key" />
        <TextField label="Operator" />
        <TextField label="Values" />
      </SegmentedField>
    );

    // Three segments produce two separators between them.
    expect(screen.getAllByRole('separator')).toHaveLength(2);
  });

  it('raises a single error state on the group with one helper line', () => {
    render(
      <SegmentedField label="Condition" error helperText="Required">
        <TextField label="Key" />
      </SegmentedField>
    );

    expect(screen.getByRole('group', { name: 'Condition' })).toHaveAttribute(
      'aria-invalid',
      'true'
    );
    expect(screen.getByText('Required')).toBeInTheDocument();
  });

  // The chrome-strip relies on this MUI internal class; if a MUI upgrade renames
  // it, this fails loudly instead of silently leaving a double border.
  it('keeps targeting the MUI outline class it strips', () => {
    render(
      <SegmentedField>
        <TextField label="Key" />
      </SegmentedField>
    );

    expect(
      document.querySelector('.MuiOutlinedInput-notchedOutline')
    ).not.toBeNull();
  });
});
