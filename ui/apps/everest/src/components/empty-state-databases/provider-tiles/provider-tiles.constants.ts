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

import type { SxProps, Theme } from '@mui/material';

export const gridSx: SxProps<Theme> = {
  display: 'flex',
  flexWrap: 'wrap',
  justifyContent: 'center',
  // Equal-height cards per row (content-driven, no fixed height).
  alignItems: 'stretch',
  gap: 2,
  width: '100%',
  maxWidth: 900,
};

export const cardSx: SxProps<Theme> = {
  // Fixed width so cards don't jump as rows reflow.
  flex: '0 1 260px',
  minWidth: 240,
  maxWidth: 260,
  position: 'relative',
  display: 'flex',
  flexDirection: 'column',
  borderRadius: (theme) => `${theme.shape.borderRadius}px`,
  border: '1px solid',
  // Let the hover extension grow past the card bottom.
  overflow: 'visible',
  borderColor: (theme) => theme.palette.dividers?.divider,
  backgroundColor: (theme) => theme.palette.surfaces?.elevation1,
  transition: (theme) =>
    theme.transitions.create(['border-color', 'box-shadow'], {
      duration: theme.transitions.duration.shorter,
    }),
  '&:hover': {
    borderColor: (theme) => theme.palette.dividers?.dividerStrong,
    boxShadow: 1,
  },
};

export const cardActionAreaSx: SxProps<Theme> = {
  height: '100%',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'stretch',
  // Drop CardActionArea's grey hover overlay; keep the focus highlight.
  '&:hover .MuiCardActionArea-focusHighlight': {
    opacity: 0,
  },
};

// Icon strip on `elevation2` — one step darker than the elevation0 page.
export const cardHeaderSx: SxProps<Theme> = {
  position: 'relative',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  py: 2.5,
  px: 2,
  backgroundColor: (theme) => theme.palette.surfaces?.elevation2,
  borderBottom: '1px solid',
  borderColor: (theme) => theme.palette.dividers?.divider,
  // Round top corners (card is overflow-visible).
  borderRadius: (theme) =>
    `${theme.shape.borderRadius}px ${theme.shape.borderRadius}px 0 0`,
};

export const headerAvatarSx: SxProps<Theme> = {
  width: 48,
  height: 48,
  textTransform: 'uppercase',
  color: (theme) => theme.palette.text.primary,
};

export const headerChipSx: SxProps<Theme> = {
  position: 'absolute',
  top: (theme) => theme.spacing(1),
  right: (theme) => theme.spacing(1),
};

export const cardContentSx: SxProps<Theme> = {
  flexGrow: 1,
  display: 'flex',
  flexDirection: 'column',
  gap: 0.75,
  py: 2,
  px: 2,
  textAlign: 'left',
};

// Anchors the extension overlay right under the header.
export const bodyAnchorSx: SxProps<Theme> = {
  position: 'relative',
  flexGrow: 1,
  display: 'flex',
  flexDirection: 'column',
};

export const descriptionSx: SxProps<Theme> = {
  color: 'text.secondary',
  display: '-webkit-box',
  WebkitLineClamp: '2',
  WebkitBoxOrient: 'vertical',
  overflow: 'hidden',
};

// Expanded description, capped so very long text can't run off the page.
export const descriptionFullSx: SxProps<Theme> = {
  color: 'text.secondary',
  display: '-webkit-box',
  WebkitLineClamp: '10',
  WebkitBoxOrient: 'vertical',
  overflow: 'hidden',
  wordBreak: 'break-word',
};

// Overlay that duplicates the card tail, continuing the side borders + rounded
// bottom so it reads as the card lengthening (mounted only while active).
export const expandedSx: SxProps<Theme> = {
  position: 'absolute',
  top: 0,
  left: -1,
  right: -1,
  // At least the card height so its footer aligns with the real one.
  minHeight: '100%',
  display: 'flex',
  flexDirection: 'column',
  backgroundColor: (theme) => theme.palette.surfaces?.elevation1,
  borderLeft: '1px solid',
  borderRight: '1px solid',
  borderBottom: '1px solid',
  borderColor: (theme) => theme.palette.dividers?.dividerStrong,
  borderRadius: (theme) =>
    `0 0 ${theme.shape.borderRadius}px ${theme.shape.borderRadius}px`,
  boxShadow: 3,
  overflow: 'hidden',
  zIndex: 3,
  '@keyframes providerTileExpand': {
    from: { opacity: 0 },
    to: { opacity: 1 },
  },
  animation: (theme) =>
    `providerTileExpand ${theme.transitions.duration.shorter}ms ease`,
};

export const categoriesRowSx: SxProps<Theme> = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: 0.5,
  mt: 0.5,
};

const footerBaseSx = {
  display: 'flex',
  alignItems: 'center',
  gap: 1,
  py: 1.25,
  px: 2,
  borderTop: '1px solid',
  borderColor: (theme: Theme) => theme.palette.dividers?.divider,
  color: 'text.secondary',
  // Round bottom corners (card is overflow-visible).
  borderRadius: (theme: Theme) =>
    `0 0 ${theme.shape.borderRadius}px ${theme.shape.borderRadius}px`,
} satisfies SxProps<Theme>;

export const cardFooterSx: SxProps<Theme> = {
  ...footerBaseSx,
  justifyContent: 'space-between',
};

// Footer variant when there's no version: keep the forward arrow right-aligned.
export const cardFooterEndSx: SxProps<Theme> = {
  ...footerBaseSx,
  justifyContent: 'flex-end',
};
