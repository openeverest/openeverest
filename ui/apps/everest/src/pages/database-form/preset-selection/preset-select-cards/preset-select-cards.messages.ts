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
import { PresetScope } from '../preset-selection.types';

export const Messages = {
  scratchTitle: 'Start from scratch',
  scratchCaption: 'Configure everything manually.',
  scope: {
    [PresetScope.Cluster]: 'Cluster-wide',
    [PresetScope.Namespace]: 'Namespace',
  },
  summary: {
    component: (name: string, fields: string) => `${name} (${fields})`,
    field: (label: string, value: string) => `${label}: ${value}`,
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
  loadError: "Presets didn't load.",
  partialLoadError: "Some presets didn't load.",
  retry: 'Retry',
  presetUnavailable: (name: string) =>
    `Preset “${name}” is no longer available.`,
  presetDroppedHint:
    'The form is back to its defaults. Pick another preset or continue from scratch.',
  presetFromOtherNamespace: (
    name: string,
    ownerNamespace: string,
    wizardNamespace: string
  ) =>
    `Preset “${name}” belongs to namespace “${ownerNamespace}” and can't be used in “${wizardNamespace}”.`,
};
