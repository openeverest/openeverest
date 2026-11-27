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
  Chip,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';
import Dialog from '../dialog';
import { Messages } from './key-value-summary.messages';
import type { KeyValueSummaryProps } from './key-value-summary.types';

// A count chip that opens the full key/value list in a dialog: read-only
// views stay compact however many entries a map holds or however long its
// keys are. The dialog is all table — the chip's row already names the field.
export const KeyValueSummary = ({
  entries,
  label,
  dataTestId = 'key-value-summary',
}: KeyValueSummaryProps) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Chip
        size="small"
        variant="outlined"
        label={Messages.entryCount(entries.length)}
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={open}
        data-testid={`${dataTestId}-toggle`}
      />
      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        fullWidth
        maxWidth="sm"
        scroll="paper"
        slotProps={{ paper: { 'aria-label': label } }}
        data-testid={`${dataTestId}-dialog`}
      >
        <Table
          size="small"
          stickyHeader
          sx={(theme) => ({
            my: 1,
            // Sticky header cells scroll over the rows; paint them like the paper.
            '& th': {
              fontWeight: 600,
              backgroundColor: theme.palette.background.paper,
            },
            // A long key or value wraps at its separators first, and inside a
            // word only when it has to.
            '& td, & th': {
              borderBottom: `1px solid ${theme.palette.divider}`,
              overflowWrap: 'anywhere',
            },
            // The floors keep one very long key or value from squeezing the
            // other column down to a letter or two per line. On a narrow
            // screen each takes at most half of what the dialog's margins and
            // the button column leave.
            '& td:first-of-type, & th:first-of-type': {
              minWidth: 'min(160px, calc(50vw - 67px))',
              pl: 3,
            },
            '& td:nth-of-type(2), & th:nth-of-type(2)': {
              minWidth: 'min(120px, calc(50vw - 67px))',
            },
            '& td:last-of-type, & th:last-of-type': {
              width: 40,
              py: 0,
              pr: 3,
            },
            '& tr:last-of-type td': { borderBottom: 0 },
          })}
        >
          <TableHead>
            <TableRow>
              <TableCell>{Messages.key}</TableCell>
              <TableCell>{Messages.value}</TableCell>
              <TableCell align="right">
                <IconButton
                  size="small"
                  aria-label={Messages.close}
                  data-testid={`${dataTestId}-dialog-close`}
                  onClick={() => setOpen(false)}
                >
                  <CloseIcon fontSize="small" />
                </IconButton>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {entries.map(({ key, value, masked }, index) => (
              // Masking can collapse keys into one string; the index keeps
              // the row key unique. The list is static, so it is safe.
              <TableRow key={`${index}:${key}`}>
                <TableCell>{key}</TableCell>
                <TableCell>{value}</TableCell>
                <TableCell align="right">
                  {!masked && (
                    <CopyToClipboardButton
                      textToCopy={`${key}=${value}`}
                      buttonProps={{
                        size: 'small',
                        'aria-label': Messages.copyEntry(key),
                        sx: { p: 0.5 },
                      }}
                      iconSx={{ fontSize: 16 }}
                    />
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Dialog>
    </>
  );
};

export default KeyValueSummary;
