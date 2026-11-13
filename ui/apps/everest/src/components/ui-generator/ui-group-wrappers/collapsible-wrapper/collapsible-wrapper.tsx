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
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Stack,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { get, useFormContext } from 'react-hook-form';
import RoundedBox from 'components/rounded-box';
import { readDefaultExpanded } from 'components/ui-generator/utils/collapsible/read-default-expanded';
import { CollapsibleWrapperProps } from './collapsible-wrapper.types';

// Bordered card whose body folds. Visual only — children stay mounted and
// keep validating / submitting whether the section is open or closed.
export const CollapsibleWrapper = ({
  children,
  label,
  description,
  groupParams,
  fieldNames = [],
}: CollapsibleWrapperProps) => {
  const [expanded, setExpanded] = useState(() =>
    readDefaultExpanded(groupParams)
  );
  const { formState } = useFormContext();
  const hasError =
    !expanded &&
    fieldNames.some((name) => {
      const error = get(formState.errors, name);
      return error != null;
    });

  return (
    <RoundedBox
      boxProps={{
        sx: {
          // Accordion owns padding via its summary/details.
          p: 0,
          overflow: 'hidden',
        },
      }}
    >
      <Accordion
        expanded={expanded}
        onChange={(_event, next) => setExpanded(next)}
        disableGutters
        elevation={0}
        sx={{
          bgcolor: 'transparent',
          boxShadow: 'none',
          '&:before': { display: 'none' },
        }}
      >
        <AccordionSummary
          expandIcon={<ExpandMoreIcon />}
          sx={{
            px: 2,
            minHeight: 48,
            '& .MuiAccordionSummary-content': {
              my: 1.5,
            },
          }}
        >
          <Box
            sx={{
              display: 'flex',
              alignItems: 'flex-start',
              gap: 1,
              pr: 1,
            }}
          >
            {hasError && (
              <ErrorOutlineIcon
                color="error"
                data-testid="collapsible-group-error"
                sx={{ mt: 0.25, flexShrink: 0 }}
              />
            )}
            <Stack sx={{ gap: 0.5 }}>
              {label && (
                <Typography variant="sectionHeading">{label}</Typography>
              )}
              {description && (
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  {description}
                </Typography>
              )}
            </Stack>
          </Box>
        </AccordionSummary>
        <AccordionDetails sx={{ px: 2, pt: 0, pb: 2 }}>
          <Stack spacing={2}>{children}</Stack>
        </AccordionDetails>
      </Accordion>
    </RoundedBox>
  );
};
