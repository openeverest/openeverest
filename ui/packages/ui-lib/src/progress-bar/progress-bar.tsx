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

import { Box, LinearProgress } from '@mui/material';
import { ProgressBarProps } from './progress-bar.types';

const ProgressBar = ({
  dataTestId,
  value,
  buffer,
  total,
  label,
}: ProgressBarProps) => {
  const value1Percentage = (value / total) * 100;
  const value2Percentage = (buffer / total) * 100;
  const isOverLimit = value2Percentage > 100;
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: '37px',
        width: '100%',
      }}
    >
      <Box
        sx={{ alignSelf: 'end', fontSize: '12px' }}
        data-testid={dataTestId ? `${dataTestId}-label` : 'progress-bar-label'}
      >
        {label}
      </Box>
      <LinearProgress
        variant="buffer"
        value={value1Percentage}
        valueBuffer={value2Percentage}
        data-testid={dataTestId ?? 'progress-bar'}
        sx={[
          {
            '&': {
              padding: '4px',
              backgroundColor: 'action.selected',
              borderRadius: '32px',
            },
            '& .MuiLinearProgress-bar': {
              margin: '1.6px',
              borderRadius: '32px',
            },
            '& .MuiLinearProgress-dashed': {
              display: 'none',
            },
          },
          isOverLimit
            ? {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar1': {
                  display: 'none',
                },
              }
            : {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar1': {
                  display: 'block',
                },
              },
          isOverLimit
            ? {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar2': {
                  backgroundColor: 'warning.main',
                },
              }
            : {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar2': {
                  backgroundColor: 'primary.contrastText',
                },
              },
          isOverLimit
            ? {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar2': {
                  transform: 'none !important',
                },
              }
            : {
                '&.MuiLinearProgress-buffer > .MuiLinearProgress-bar2': {
                  transform: null,
                },
              },
        ]}
      />
    </Box>
  );
};

export default ProgressBar;
