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

import { capitalize, Typography, TypographyProps } from '@mui/material';
import StatusField from 'components/status-field';
import { COMPONENT_STATUS, CONTAINER_STATUS } from '../components.constants';
import {
  BaseStatus,
  StatusFieldProps,
} from 'components/status-field/status-field.types';

const ComponentStatus = <T extends COMPONENT_STATUS | CONTAINER_STATUS>({
  status,
  statusMap,
  typographyProps,
}: {
  status: T;
  statusMap: Record<T, BaseStatus>;
  typographyProps?: TypographyProps;
} & StatusFieldProps<T>) => (
  <StatusField status={status} statusMap={statusMap}>
    <Typography
      variant="body2"
      {...typographyProps}
      sx={[
        {
          fontWeight: 'bold',
        },
        ...(typographyProps?.sx
          ? Array.isArray(typographyProps.sx)
            ? typographyProps.sx
            : [typographyProps.sx]
          : []),
      ]}
    >
      {capitalize(status)}
    </Typography>
  </StatusField>
);

export default ComponentStatus;
