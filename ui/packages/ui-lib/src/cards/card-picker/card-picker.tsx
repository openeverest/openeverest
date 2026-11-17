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

import { useMemo, useRef, useState } from 'react';
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  TextField,
  Typography,
} from '@mui/material';
import { CardPickerItem, CardPickerProps } from './card-picker.types';
import {
  DEFAULT_MIN_CARD_PX,
  GRID_GAP_PX,
  buildGridSx,
} from './card-picker.constants';
import { Messages as defaultMessages } from './card-picker.messages';
import {
  getSearchTokens,
  itemMatches,
  withSelectedPinned,
} from './card-picker.utils';
import { useGridColumnCount } from './use-grid-column-count';
import { SelectableCard } from './selectable-card';
import { BrowseCard } from './browse-card';

export const CardPicker = ({
  items,
  selectedId,
  onSelect,
  leadCard,
  error,
  statusCard,
  messages,
  minCardPx = DEFAULT_MIN_CARD_PX,
  sx,
}: CardPickerProps) => {
  const msg = { ...defaultMessages, ...messages };
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const gridRef = useRef<HTMLDivElement>(null);
  const columns = useGridColumnCount(gridRef, minCardPx, GRID_GAP_PX);
  const gridSx = useMemo(() => buildGridSx(minCardPx), [minCardPx]);

  const tokens = useMemo(() => getSearchTokens(query), [query]);
  const filtered = useMemo(
    () =>
      tokens.length > 0 ? items.filter((it) => itemMatches(it, tokens)) : items,
    [items, tokens]
  );

  const select = (id: string) => {
    onSelect(id);
    setOpen(false);
    setQuery('');
  };

  const closeDialog = () => {
    setOpen(false);
    setQuery('');
  };

  const renderCard = (item: CardPickerItem, highlight: boolean) => (
    <SelectableCard
      key={item.id}
      item={item}
      selected={selectedId === item.id}
      tokens={highlight ? tokens : []}
      onSelect={() => select(item.id)}
    />
  );

  const leadCount = leadCard ? 1 : 0;
  const allFitInline = leadCount + items.length <= columns;
  const inlineItems = allFitInline
    ? items
    : withSelectedPinned(
        items,
        Math.max(0, columns - leadCount - 1),
        selectedId
      );

  return (
    <Box sx={sx}>
      <Box ref={gridRef} sx={gridSx}>
        {leadCard && renderCard(leadCard, false)}
        {inlineItems.map((it) => renderCard(it, false))}
        {!allFitInline && (
          <BrowseCard
            label={msg.browseAll(items.length)}
            onClick={() => setOpen(true)}
          />
        )}
        {statusCard && (
          <SelectableCard
            disabled
            role="status"
            item={{
              id: '',
              title: statusCard.title,
              subtitle: statusCard.subtitle,
            }}
            selected={false}
            tokens={[]}
            onSelect={() => {}}
          />
        )}
      </Box>
      {error && (
        <Typography
          variant="caption"
          color="error"
          sx={{ mt: 1, display: 'block' }}
        >
          {error}
        </Typography>
      )}

      <Dialog open={open} onClose={closeDialog} maxWidth="md" fullWidth>
        <DialogTitle>{msg.dialogTitle}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            size="small"
            fullWidth
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={msg.searchPlaceholder}
            inputProps={{ 'aria-label': msg.searchAriaLabel }}
            sx={{ mb: 2, mt: 1 }}
          />
          <Box sx={gridSx}>
            {leadCard && renderCard(leadCard, false)}
            {filtered.map((it) => renderCard(it, true))}
          </Box>
          {tokens.length > 0 && filtered.length === 0 && (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
              {msg.noMatches(query)}
            </Typography>
          )}
        </DialogContent>
      </Dialog>
    </Box>
  );
};
