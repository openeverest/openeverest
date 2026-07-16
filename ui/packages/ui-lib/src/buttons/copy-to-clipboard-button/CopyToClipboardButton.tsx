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
import { Button, IconButton } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import { CopyToClipboardButtonProps } from './CopyToClipboardButton.types';
import { Messages } from './clipboard.messages';

const CopyToClipboardButton = ({
  textToCopy,
  buttonProps,
  iconSx,
  showCopyButtonText,
  copyCommand = 'Copy command',
}: CopyToClipboardButtonProps) => {
  const [open, setOpen] = useState(false);
  const clipboardAvailable = !!navigator.clipboard;

  const handleClick = () => {
    if (clipboardAvailable) {
      navigator.clipboard.writeText(textToCopy);
      setOpen(true);

      setTimeout(() => {
        setOpen(false);
      }, 1500);
    }
  };

  return (
    <Tooltip
      {...(clipboardAvailable && {
        disableHoverListener: true,
        open,
      })}
      disableFocusListener
      disableTouchListener
      title={clipboardAvailable ? Messages.copied : Messages.restrictedAccess}
      slotProps={{
        popper: {
          disablePortal: true,
        },
      }}
    >
      {showCopyButtonText ? (
        <Button
          sx={{ ...buttonProps?.sx, display: 'flex', gap: 1 }}
          onClick={handleClick}
          disabled={!clipboardAvailable}
          {...buttonProps}
        >
          <ContentCopyOutlinedIcon sx={iconSx} />
          {copyCommand}
        </Button>
      ) : (
        <span>
          <IconButton
            component="div"
            sx={{
              ...buttonProps?.sx,
              '&.Mui-disabled': {
                pointerEvents: 'auto',
              },
            }}
            onClick={handleClick}
            disabled={!clipboardAvailable}
            {...buttonProps}
          >
            <ContentCopyOutlinedIcon sx={iconSx} />
          </IconButton>
        </span>
      )}
    </Tooltip>
  );
};
export default CopyToClipboardButton;
