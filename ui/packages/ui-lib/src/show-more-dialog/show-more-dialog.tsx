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
import {
  Button,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
} from '@mui/material';
import Dialog from '../dialog';
import { SHOW_MORE_LINK_SX } from './show-more-dialog.constants';
import { Messages } from './show-more-dialog.messages';
import type { ShowMoreDialogProps } from './show-more-dialog.types';

// A "Show more" link that opens its content in a dialog.
export const ShowMoreDialog = ({
  dialogTitle,
  children,
  dataTestId = 'show-more',
  linkLabel = Messages.showMore,
  closeLabel = Messages.close,
  linkTypographyProps,
  dialogProps,
}: ShowMoreDialogProps) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Link
        component="button"
        type="button"
        underline="always"
        variant={linkTypographyProps?.variant ?? 'caption'}
        color={linkTypographyProps?.color}
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={open}
        data-testid={`${dataTestId}-toggle`}
        sx={{ ...SHOW_MORE_LINK_SX, ...linkTypographyProps?.sx }}
      >
        {linkLabel}
      </Link>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        fullWidth
        maxWidth="md"
        scroll="paper"
        {...dialogProps}
      >
        <DialogTitle>{dialogTitle}</DialogTitle>
        <DialogContent sx={{ pt: 1 }}>{children}</DialogContent>
        <DialogActions>
          <Button
            variant="text"
            onClick={() => setOpen(false)}
            data-testid={`${dataTestId}-dialog-close`}
          >
            {closeLabel}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default ShowMoreDialog;
