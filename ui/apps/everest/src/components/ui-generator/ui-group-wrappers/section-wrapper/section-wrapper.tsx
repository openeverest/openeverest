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

import { Box, Typography } from '@mui/material';
import RoundedBox from 'components/rounded-box';
import { SectionWrapperProps } from './section-wrapper.types';

const SectionWrapper = ({
  label,
  description,
  disabled = false,
  children,
}: SectionWrapperProps) => {
  return (
    <RoundedBox
      title={label}
      boxProps={{
        sx: {
          opacity: disabled ? 0.5 : 1,
          pointerEvents: disabled ? 'none' : undefined,
        },
      }}
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
