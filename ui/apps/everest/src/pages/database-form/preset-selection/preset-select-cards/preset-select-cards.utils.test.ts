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

import { PRESET_NAME, TEAM_A, TEAM_B, teamAPreset } from '../__mocks__/presets';
import {
  getDroppedPresetMessage,
  presetToCardItem,
} from './preset-select-cards.utils';

describe('presetToCardItem', () => {
  it('maps the id/title and the displayed scope, meta + resources', () => {
    const item = presetToCardItem(teamAPreset);

    // The id (not the name) is what the picker selects by: names can repeat.
    expect(item.id).toBe(teamAPreset.id);
    expect(item.title).toBe(PRESET_NAME);
    expect(item.subtitle).toBe('Namespace · replicaSet · 7.0.18');
    expect(item.detail).toBe('3 nodes · 1 CPU · 4Gi RAM · 25Gi disk');
  });

  it('builds searchText from the whole spec, including fields not shown', () => {
    const haystack =
      presetToCardItem(teamAPreset).searchText?.toLowerCase() ?? '';

    // providerRef.name is searchable but is never rendered on the card.
    expect(haystack).toContain('percona-server-mongodb');
    expect(haystack).toContain('providerref');
    expect(haystack).toContain('storage');
    // The scope label is searchable, to narrow the list to one kind.
    expect(haystack).toContain('namespace');
  });
});

describe('getDroppedPresetMessage', () => {
  // The new namespace may have its own preset of that name, so the message
  // says which namespace the dropped one belongs to.
  it('names the owning namespace after a namespace switch', () => {
    expect(
      getDroppedPresetMessage({
        name: PRESET_NAME,
        wizardNamespace: TEAM_B,
        ownerNamespace: TEAM_A,
      })
    ).toBe(
      "Preset “psmdb-default” belongs to namespace “team-a” and can't be used in “team-b”."
    );
  });

  it('says a namespace preset is gone when it disappeared in place', () => {
    expect(
      getDroppedPresetMessage({
        name: PRESET_NAME,
        wizardNamespace: TEAM_A,
        ownerNamespace: TEAM_A,
      })
    ).toBe('Preset “psmdb-default” is no longer available.');
  });

  it('says a cluster preset is gone, whatever the namespace', () => {
    expect(
      getDroppedPresetMessage({ name: PRESET_NAME, wizardNamespace: TEAM_B })
    ).toBe('Preset “psmdb-default” is no longer available.');
  });
});
