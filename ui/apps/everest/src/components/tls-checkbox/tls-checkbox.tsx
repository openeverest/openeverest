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

import { Box, FormControlLabel, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { CheckboxInput } from '@percona/ui-lib';
import { TlsCheckboxProps } from './tls-checkbox.types';
import { Messages } from './tls-checkbox.messages';

const TlsCheckbox = ({ formControlLabelProps }: TlsCheckboxProps) => (
  <FormControlLabel
    {...formControlLabelProps}
    label={
      <Box
        sx={{
          display: 'flex',
          mt: 0,
        }}
      >
        {Messages.verifyTLS}
        <Tooltip
          title={Messages.tooltip}
          arrow
          placement="right"
          sx={{ ml: 1 }}
        >
          <InfoOutlinedIcon />
        </Tooltip>
      </Box>
    }
    control={<CheckboxInput name="verifyTLS" />}
  />
);

export default TlsCheckbox;
