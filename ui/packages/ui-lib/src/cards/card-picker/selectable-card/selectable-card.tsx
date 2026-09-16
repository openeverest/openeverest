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

import { Card, CardActionArea, CardContent, Typography } from '@mui/material';
import { CardPickerItem } from '../card-picker.types';
import { Highlight } from '../highlight';

interface SelectableCardProps {
  item: CardPickerItem;
  selected: boolean;
  tokens: string[];
  onSelect: () => void;
}

export const SelectableCard = ({
  item,
  selected,
  tokens,
  onSelect,
}: SelectableCardProps) => (
  <Card
    variant="selectable"
    className={selected ? 'selected' : undefined}
    sx={{ display: 'flex', flexDirection: 'column' }}
  >
    <CardActionArea
      onClick={onSelect}
      aria-pressed={selected}
      sx={{ height: '100%' }}
    >
      <CardContent
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 0.75,
          py: 2,
          px: 2,
          textAlign: 'left',
        }}
      >
        <Typography variant="sectionHeading">
          <Highlight text={item.title} tokens={tokens} />
        </Typography>
        {item.subtitle && (
          <Typography variant="body2" color="text.secondary">
            <Highlight text={item.subtitle} tokens={tokens} />
          </Typography>
        )}
        {item.detail && (
          <Typography variant="helperText" color="text.secondary">
            <Highlight text={item.detail} tokens={tokens} />
          </Typography>
        )}
      </CardContent>
    </CardActionArea>
  </Card>
);
