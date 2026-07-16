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

import { Stack } from '@mui/material';
import { UnknownIcon } from '@percona/ui-lib';
import { BaseStatus, StatusFieldProps } from './status-field.types';
import { STATUS_TO_ICON } from './status-field.utils';

function StatusField<T extends string | number | symbol>({
  status,
  statusMap,
  children,
  dataTestId,
  iconProps,
  stackProps,
  defaultIcon = UnknownIcon,
}: StatusFieldProps<T>) {
  const mappedStatus: BaseStatus = statusMap[status];
  const MappedIcon = STATUS_TO_ICON[mappedStatus] || defaultIcon;

  return (
    <Stack
      direction="row"
      data-testid={`${dataTestId ? `${dataTestId}-` : ''}status`}
      {...stackProps}
      sx={[
        {
          gap: 1,
          height: 'fit-content',
          alignItems: 'center',
        },
        ...(stackProps?.sx
          ? Array.isArray(stackProps.sx)
            ? stackProps.sx
            : [stackProps.sx]
          : []),
      ]}
    >
      <MappedIcon {...iconProps} />
      {children}
    </Stack>
  );
}

export default StatusField;
