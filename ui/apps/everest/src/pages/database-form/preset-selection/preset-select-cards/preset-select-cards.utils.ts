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

import { CardPickerItem, flattenToSearchText } from '@percona/ui-lib';
import { InstancePreset } from 'shared-types/api.types';
import { Messages } from './preset-select-cards.messages';

const toText = (value: number | string | undefined): string | undefined =>
  value === undefined || value === null ? undefined : String(value);

const getMeta = (preset: InstancePreset): string =>
  [preset.spec?.topology?.type, preset.spec?.version]
    .filter(Boolean)
    .join(' · ');

const getResources = (preset: InstancePreset): string => {
  const engine = preset.spec?.components?.engine;
  const limits = engine?.resources?.limits;
  const nodes = engine?.replicas;
  const cpu = toText(limits?.cpu);
  const memory = toText(limits?.memory);
  const disk = toText(engine?.storage?.size);

  return [
    nodes !== undefined ? Messages.summary.nodes(nodes) : undefined,
    cpu && Messages.summary.cpu(cpu),
    memory && Messages.summary.memory(memory),
    disk && Messages.summary.disk(disk),
  ]
    .filter(Boolean)
    .join(' · ');
};

export const presetToCardItem = (preset: InstancePreset): CardPickerItem => {
  const name = preset.metadata?.name ?? '';
  const subtitle = getMeta(preset);
  const detail = getResources(preset);
  return {
    id: name,
    title: name,
    subtitle,
    detail,
    // Search the whole spec (e.g. storageClass) — not just the displayed fields.
    searchText: `${name} ${flattenToSearchText(preset.spec)}`,
  };
};
