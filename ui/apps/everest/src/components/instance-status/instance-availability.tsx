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

import { Box, Tooltip } from '@mui/material';
import { InstanceAvailability as Availability } from 'shared-types/instance.types';
import { Messages } from 'utils/instance-status.messages';

interface InstanceAvailabilityProps {
  available: Availability;
  dataTestId?: string;
}

// Whether clients can connect, kept apart from the status: a database can be
// up while something is wrong with it, and down while nothing is.
export const InstanceAvailability = ({
  available,
  dataTestId,
}: InstanceAvailabilityProps) => (
  <Tooltip title={Messages.availabilityHint[available]} placement="right" arrow>
    <Box
      component="span"
      data-testid={`${dataTestId ? `${dataTestId}-` : ''}available`}
    >
      {Messages.availability[available]}
    </Box>
  </Tooltip>
);
