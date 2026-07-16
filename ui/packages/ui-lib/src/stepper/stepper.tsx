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

import { Fragment } from 'react';
import { Stepper as MuiStepper, useTheme } from '@mui/material';
import { StepperProps } from './stepper.types';

const Stepper = ({
  noConnector,
  connector,
  dataTestId,
  sx,
  ...props
}: StepperProps) => {
  const theme = useTheme();

  return (
    <MuiStepper
      data-testid={dataTestId}
      sx={[
        {
          '.MuiStepIcon-root.Mui-active': {
            color: theme.palette.text.primary,
          },
          '.MuiStepIcon-root.Mui-completed': {
            color: theme.palette.primary.light,
          },
          '.MuiStepLabel-label.Mui-active': {
            color: theme.palette.text.primary,
            fontWeight: 600,
          },
          '.MuiStepLabel-label.Mui-completed': {
            color: theme.palette.text.secondary,
            fontWeight: 600,
          },
          '.MuiStepLabel-label': {
            fontWeight: 600,
          },
        },
        noConnector && {
          '.MuiStep-root': {
            padding: 0,
          },
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      {...props}
      connector={noConnector ? <Fragment /> : connector}
    />
  );
};

export default Stepper;
