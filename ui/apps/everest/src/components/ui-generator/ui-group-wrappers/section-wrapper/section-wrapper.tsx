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

import { Box, BoxProps, Typography } from '@mui/material';
import RoundedBox from 'components/rounded-box';
import { SectionWrapperProps } from './section-wrapper.types';

const SectionWrapper = ({
  fieldName,
  label,
  description,
  disabled = false,
  controlComponent,
  'data-testid': dataTestId,
  children,
}: SectionWrapperProps) => {
  const header = label ? (
    <Box display="flex" justifyContent="space-between" alignItems="center">
      <Typography variant="sectionHeading">{label}</Typography>
      {controlComponent && <Box>{controlComponent}</Box>}
    </Box>
  ) : undefined;

  return (
    <RoundedBox
      title={header}
      boxProps={
        {
          'data-testid':
            dataTestId ?? (fieldName ? `section-${fieldName}` : undefined),
          sx: {
            opacity: disabled ? 0.5 : 1,
            pointerEvents: disabled ? 'none' : undefined,
          },
        } as BoxProps
      }
    >
      {description && (
        <Typography
          variant="caption"
          color="text.secondary"
          display="block"
          sx={{ mt: 0.5, mb: 1.5 }}
        >
          {description}
        </Typography>
      )}
      <Box>{children}</Box>
    </RoundedBox>
  );
};

export default SectionWrapper;
