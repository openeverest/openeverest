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

import { Avatar, Box, Tooltip, Typography } from '@mui/material';
import HelpOutlineIcon from '@mui/icons-material/HelpOutlineOutlined';
import type { ProviderIdentityProps } from './provider-identity.types';

const avatarSx = {
  width: 36,
  height: 36,
  flexShrink: 0,
  fontSize: '1rem',
  textTransform: 'uppercase',
  color: (theme: { palette: { text: { primary: string } } }) =>
    theme.palette.text.primary,
};

const ellipsisLabelSx = {
  flexGrow: 1,
  minWidth: 0,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
};

const helperIconSx = { flexShrink: 0, color: 'text.secondary' };

// Presentational atom that renders a provider's identity: avatar (only when
// catalog metadata is available), label, and an optional description helper
// icon. Composed by ProviderTiles (grid) and CreateDbButton drawer rows.
export const ProviderIdentity = ({
  label,
  meta,
  showDescription = false,
  ellipsis = false,
  labelVariant = 'body1',
}: ProviderIdentityProps) => {
  const initial = label.charAt(0);
  const showTooltip = showDescription && Boolean(meta?.description);

  return (
    <>
      {meta && (
        <Avatar
          src={meta.icon}
          variant="rounded"
          sx={{
            ...avatarSx,
            bgcolor: meta.icon ? 'transparent' : 'action.selected',
          }}
        >
          {initial}
        </Avatar>
      )}
      <Typography
        variant={labelVariant}
        sx={ellipsis ? ellipsisLabelSx : { flexGrow: 1, textAlign: 'left' }}
      >
        {label}
      </Typography>
      {showTooltip && (
        <Tooltip title={meta?.description} placement="top" arrow>
          <Box
            component="span"
            tabIndex={0}
            aria-label={meta?.description}
            sx={{ display: 'inline-flex', cursor: 'help' }}
            // Prevent the tooltip icon from triggering navigation on the
            // wrapping CardActionArea / Link.
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
          >
            <HelpOutlineIcon
              fontSize="small"
              aria-hidden="true"
              sx={helperIconSx}
            />
          </Box>
        </Tooltip>
      )}
    </>
  );
};
