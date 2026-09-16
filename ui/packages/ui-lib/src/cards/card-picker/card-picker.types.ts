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

import { SxProps, Theme } from '@mui/material';

export interface CardPickerItem {
  id: string;
  title: string;
  subtitle?: string;
  detail?: string;
  // Optional precomputed lowercase haystack; defaults to title + subtitle + detail.
  searchText?: string;
}

export interface CardPickerMessages {
  searchPlaceholder: string;
  searchAriaLabel: string;
  browseAll: (count: number) => string;
  dialogTitle: string;
  noMatches: (query: string) => string;
}

export interface CardPickerProps {
  items: CardPickerItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  // Always-first card (e.g. a "start from scratch" / "none" option). Selected
  // when selectedId === leadCard.id.
  leadCard?: CardPickerItem;
  loading?: boolean;
  error?: string;
  messages?: Partial<CardPickerMessages>;
  minCardPx?: number;
  sx?: SxProps<Theme>;
}
