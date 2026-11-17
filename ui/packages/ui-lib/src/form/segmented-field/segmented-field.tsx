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

import { Children, Fragment, isValidElement } from 'react';
import { Box, Divider, FormHelperText } from '@mui/material';
import { SegmentedFieldProps } from './segmented-field.types';

// Keep the label in the accessibility tree but out of sight (the container is
// the visible frame, not each child).
const srOnlyLabelSx = {
  position: 'absolute',
  width: '1px',
  height: '1px',
  padding: 0,
  margin: '-1px',
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  whiteSpace: 'nowrap',
} as const;

// Strip each child field's own frame (outline / underline / floating label /
// helper / inner focus) so only the container shows. This deliberately targets
// MUI internal classes; the test asserts these classes still exist so a MUI
// upgrade that renames them fails loudly instead of silently un-styling fields.
const stripChildChromeSx = {
  '& .MuiFormControl-root': { m: 0 },
  '& .MuiInputBase-root': { borderRadius: 0 },
  // !important so MUI's higher-specificity focus/hover/error outline rules can't
  // bring the inner border back — the container is the only frame.
  '& .MuiOutlinedInput-notchedOutline': { border: '0 !important' },
  '& .MuiInput-root::before, & .MuiInput-root::after': { display: 'none' },
  '& .MuiInputLabel-root': srOnlyLabelSx,
  '& .MuiFormHelperText-root': { display: 'none' },
};

// A single outlined frame that visually merges a few lean form controls into one
// field, with vertical dividers between them and one shared focus/error state.
const SegmentedField = ({
  children,
  label,
  error = false,
  disabled = false,
  helperText,
  dividers = false,
  sx,
}: SegmentedFieldProps) => {
  const segments = Children.toArray(children).filter(isValidElement);

  return (
    <Box sx={sx}>
      <Box
        role="group"
        aria-label={label}
        aria-invalid={error || undefined}
        aria-disabled={disabled || undefined}
        sx={[
          {
            display: 'flex',
            alignItems: 'stretch',
            border: '1px solid',
            borderColor: error ? 'error.main' : 'divider',
            borderRadius: 1,
            overflow: 'hidden',
            ...(disabled && { opacity: 0.6, pointerEvents: 'none' }),
            // Match the standard outlined field focus (dark border), not blue.
            '&:focus-within': error
              ? {}
              : {
                  borderColor: 'text.primary',
                  boxShadow: (theme) =>
                    `0 0 0 1px ${theme.palette.text.primary}`,
                },
          },
          stripChildChromeSx,
        ]}
      >
        {segments.map((segment, index) => (
          <Fragment key={segment.key ?? index}>
            {dividers && index > 0 && (
              <Divider orientation="vertical" flexItem />
            )}
            {segment}
          </Fragment>
        ))}
      </Box>
      {/* Always render the helper line so an appearing error doesn't shift the
          layout below. */}
      <FormHelperText error={error} sx={{ mx: 1.5, minHeight: '1.25rem' }}>
        {helperText}
      </FormHelperText>
    </Box>
  );
};

export default SegmentedField;
