// everest
// Copyright (C) 2023 Percona LLC
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
import { Box, IconButton, Typography } from '@mui/material';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import { Messages } from './HiddenPasswordToggle.messages';
import { CopyToClipboardButton } from '@percona/ui-lib';

type HideRowProps = {
  value: string;
  fixedAsteriskLength?: boolean;
  showCopy?: boolean;
};
export const HiddenPasswordToggle = ({
  value,
  fixedAsteriskLength = true,
  showCopy = false,
}: HideRowProps) => {
  const [show, setShow] = useState(false);
  const formattedValue = show
    ? value
    : fixedAsteriskLength
      ? Messages.asteriskHiddenText
      : value.replace(/./g, '*');

  const toggle = () => {
    setShow((prevState) => !prevState);
  };

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'flex-start',
      }}
      data-testid="hidden-row"
    >
      <Typography
        variant="body2"
        sx={[
          {
            wordBreak: 'break-all',
            pr: 1.5,
          },
          show
            ? {
                pt: 0,
              }
            : {
                pt: 0.5,
              },
        ]}
      >
        {formattedValue}
      </Typography>
      <IconButton
        onClick={toggle}
        size="small"
        aria-label={`visibility-${show ? 'off' : 'on'}`}
        sx={{ mt: -0.5 }}
      >
        {
          //TODO 1230 discussion with Nuna about common usage of this part in overview and expanded row
          show ? (
            <VisibilityOutlinedIcon color="primary" />
          ) : (
            <VisibilityOffOutlinedIcon color="primary" />
          )
        }
      </IconButton>
      {showCopy && (
        <CopyToClipboardButton
          buttonProps={{ sx: { mt: -0.5 }, color: 'primary', size: 'small' }}
          textToCopy={value}
        />
      )}
    </Box>
  );
};
