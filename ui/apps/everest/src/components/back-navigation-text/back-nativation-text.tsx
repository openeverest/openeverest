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

import { Box, IconButton, Stack, Typography } from '@mui/material';
import ArrowBackIosIcon from '@mui/icons-material/ArrowBackIos';
import { BackNavigationTextProps } from './back-nativation-text.types';

const BackNavigationText = ({
  text,
  onBackClick,
  stackProps,
  rightSlot,
}: BackNavigationTextProps) => (
  <Stack
    {...stackProps}
    sx={[
      {
        gap: 1,
        alignItems: 'center',
        flexDirection: 'row',
      },
      ...(stackProps?.sx
        ? Array.isArray(stackProps.sx)
          ? stackProps.sx
          : [stackProps.sx]
        : []),
    ]}
  >
    <IconButton onClick={onBackClick}>
      <ArrowBackIosIcon sx={{ pl: '10px' }} fontSize="large" />
    </IconButton>
    <Typography variant="h4">{text}</Typography>
    {rightSlot && <Box sx={{ ml: 'auto' }}>{rightSlot}</Box>}
  </Stack>
);

export default BackNavigationText;
