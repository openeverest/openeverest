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

import { pluralize } from '@percona/utils';

export const Messages = {
  scratchTitle: 'Start from scratch',
  scratchCaption: 'Configure everything manually.',
  summary: {
    nodes: (count: number) => pluralize(count, 'node'),
    cpu: (value: string) => `${value} CPU`,
    memory: (value: string) => `${value} RAM`,
    disk: (value: string) => `${value} disk`,
  },
  searchPlaceholder: 'Search presets by name or specs…',
  searchAriaLabel: 'Search presets by name or specs',
  browseAll: (count: number) => `Browse all ${count} presets`,
  dialogTitle: 'Choose a preset',
  noMatches: (query: string) => `No presets match “${query}”.`,
  loadErrorTitle: "Presets didn't load",
  loadErrorSubtitle: 'Check your connection and try again.',
};
